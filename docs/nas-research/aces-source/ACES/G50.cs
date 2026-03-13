using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES File Validation Error
    /// </summary>
    public class G50
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G50Detail> _detail = new List<G50Detail>();
        public G50Header Header { get; set; }
        public G50Trailer Trailer { get; set; }
        public List<G50Detail> Detail
        {
            get
            {
                return _detail;
            }
            private set
            {
                _detail = value;
            }
        }

        public G50()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G50Header();
            Trailer = new G50Trailer();
        }

        public static G50 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G50 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G50 rtn = new G50();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G50Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "FEP00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G50Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G50";
                            err.System = "ACES";
                            err.ErrorDateTime = DateTime.Now;
                            err.FilePath = rtn.TransmissionInfo.LocalFile.FullName;
                            err.Detail = line;
                            err.Active = true;

                            Utils.AddErrorEntry(err);
                            if (!movedToPending && moveOnError)
                            {
                                Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
                                movedToPending = true;
                            }
                        }
                        continue;
                    case "FEP01":
                        G50Detail d = null;
                        try
                        {
                            d = G50Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Detail)";
                            err.Description = "Missing required information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G92";
                            err.System = "ACES";
                            err.ErrorDateTime = DateTime.Now;
                            err.FilePath = rtn.TransmissionInfo.LocalFile.FullName;
                            err.Detail = line;
                            err.Active = true;

                            Utils.AddErrorEntry(err);
                            if (!movedToPending && moveOnError)
                            {
                                Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
                                movedToPending = true;
                            }
                            continue;
                        }
                        rtn.Detail.Add(d);
                        detailCount++;
                        continue;
                    case Utils.EOF:
                        if (hasTrl) continue;
                        hasTrl = true;
                        try
                        {
                            rtn.Trailer = G50Trailer.Load(line);
                        }
                        catch (RecordCountMismatch rcmEx)
                        {
                            Error err = new Error();
                            err.Message = rcmEx.Message + " (Trailer)";
                            err.Description = string.Format("Header count: {0}\tTrailer count: {1}\tActual count: {2}",
                                rtn.Header.TotalRecordCount.Value - 2,
                                rtn.Trailer.TransmitRecordCount.Value,
                                rtn.Detail.Count);
                            err.Code = "ACES_RECORD_COUNT_MISMATCH";
                            err.EdiSet = "G50";
                            err.System = "ACES";
                            err.ErrorDateTime = DateTime.Now;
                            err.FilePath = rtn.TransmissionInfo.LocalFile.FullName;
                            err.Detail = line;
                            if (!movedToPending && moveOnError)
                            {
                                Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
                                movedToPending = true;
                            }
                        }
                        break;
                }
                break;
            }

            return rtn;
        }

        public bool Process(ATLDbDataContext db)
        {
            DateTime creation = CreatedDateTime;

            List<ACES_G50> newG50s = new List<ACES_G50>(Detail.Count);
            foreach (G50Detail det in Detail)
            {
                int errCount = 0;
                try
                {
                    errCount = (from p in db.ACES_G50s
                                where p.FileName.Trim().Equals(det.FileNameToResend.Value) &&
                                p.ICLErrorCode.Trim().Equals(det.RejectCode.Value) &&
                                p.ICLErrorCodeDesc.Trim().Equals(det.RejectExplanation.Value)
                                select p).Count();
                }
                catch { }

                if (errCount < 1)
                {
                    ACES_G50 g50 = new ACES_G50();
                    g50.FileName = det.FileNameToResend.Value;
                    g50.ICLErrorCode = det.RejectCode.Value;
                    g50.ICLErrorCodeDesc = det.RejectExplanation.Value;

                    newG50s.Add(g50);
                }
            }

            ACES_G50 dbG50 = null;
            try
            {
                dbG50 = ((from p in db.ACES_G50s
                                   orderby p.ACES_G50Id descending
                                   select p).FirstOrDefault() as ACES_G50);


                int startId = 1;
                if (dbG50 != null)
                    startId = dbG50.ACES_G50Id + 1;

                foreach (ACES_G50 g in newG50s)
                {
                    g.ACES_G50Id = startId++;
                }

            }
            catch (InvalidOperationException ioex)
            {
                if (ioex.Message.Equals("Sequence contains no elements"))
                {
                    int newIds = 1;
                    foreach (ACES_G50 g in newG50s)
                    {
                        g.ACES_G50Id = newIds++;
                    }
                }
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }

            try
            {
                db.ACES_G50s.InsertAllOnSubmit(newG50s);
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
            }
            catch (Exception ex) 
            { 
                AppLog.WriteExceptionToLog(ex, null, true);
                return false;
            }

            return true;
        }
    }

    public class G50Header
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> SenderID { get; set; }
        public FixedPositionItem<string> ReceiverID { get; set; }
        public FixedPositionItem<string> TransmissionID { get; set; }
        public FixedPositionItem<DateTime> TransmissionDate { get; set; }
        public FixedPositionItem<DateTime> TransmissionTime { get; set; }
        public FixedPositionItem<string> PortCode { get; set; }
        public FixedPositionItem<string> CustomerCode { get; set; }
        public FixedPositionItem<int> TotalRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; private set; }

        public G50Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "FEP00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G50", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Required = true };
            PortCode = new FixedPositionItem<string>() { Offset = 28, Length = 2, Value = string.Empty };
            CustomerCode = new FixedPositionItem<string>() { Offset = 30, Length = 10, Value = string.Empty, Required = true };
            TotalRecordCount = new FixedPositionItem<int>() { Offset = 40, Length = 6, Value = 0, Format = "{0:000000}", Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 46, Length = 304, Value = string.Empty };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                SenderID.ToString() +
                ReceiverID.ToString() +
                TransmissionID.ToString() +
                TransmissionDate.ToString() +
                TransmissionTime.ToString() +
                PortCode.ToString() +
                CustomerCode.ToString() +
                TotalRecordCount.ToString() +
                Filler.ToString();
        }

        public static G50Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G50Header rtn = new G50Header();

            rtn.RecordID.Value = headerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.SenderID.Value = headerLine.Substring(rtn.SenderID.Offset, rtn.SenderID.Length).Trim();
            rtn.ReceiverID.Value = headerLine.Substring(rtn.ReceiverID.Offset, rtn.ReceiverID.Length).Trim();
            rtn.TransmissionID.Value = headerLine.Substring(rtn.TransmissionID.Offset, rtn.TransmissionID.Length).Trim();
            string transmissionDateTimeString =
                headerLine.Substring(rtn.TransmissionDate.Offset,
                (rtn.TransmissionDate.Length + rtn.TransmissionTime.Length)).Trim();
            DateTime tdt = DateTime.ParseExact(transmissionDateTimeString, "yyyyMMddHHmmss", CultureInfo.InvariantCulture);
            rtn.TransmissionDate.Value = tdt;
            rtn.TransmissionTime.Value = tdt;
            rtn.PortCode.Value = headerLine.Substring(rtn.PortCode.Offset, rtn.PortCode.Length).Trim();
            rtn.CustomerCode.Value = headerLine.Substring(rtn.CustomerCode.Offset, rtn.CustomerCode.Length).Trim();
            int trc;
            int.TryParse(headerLine.Substring(rtn.TotalRecordCount.Offset, rtn.TotalRecordCount.Length).Trim(), out trc);
            rtn.TotalRecordCount.Value = trc;

            return rtn;
        }
    }

    public class G50Detail : IComparable<G50Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> FileNameToResend { get; set; }
        public FixedPositionItem<string> DotSeparator { get; set; }
        public FixedPositionItem<string> FileNameExtension { get; set; }
        public FixedPositionItem<string> Filler1 { get; set; }
        public FixedPositionItem<string> RejectCode { get; set; }
        public FixedPositionItem<string> Filler2 { get; set; }
        public FixedPositionItem<string> RejectExplanation { get; set; }
        public FixedPositionItem<string> Filler3 { get; set; }
        public bool DoInsert { get; set; }

        public G50Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "FEP01", Required = true };
            FileNameToResend = new FixedPositionItem<string>() { Offset = 5, Length = 10, Value = string.Empty, Required = true };
            DotSeparator = new FixedPositionItem<string>() { Offset = 15, Length = 1, Value = string.Empty, Required = true };
            FileNameExtension = new FixedPositionItem<string>() { Offset = 16, Length = 3, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 19, Length = 2, Value = string.Empty };
            RejectCode = new FixedPositionItem<string>() { Offset = 21, Length = 8, Value = string.Empty, Required = true };
            Filler2 = new FixedPositionItem<string>() { Offset = 29, Length = 2, Value = string.Empty };
            RejectExplanation = new FixedPositionItem<string>() { Offset = 31, Length = 37, Value = string.Empty, Required = true };
            Filler3 = new FixedPositionItem<string>() { Offset = 68, Length = 282, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                FileNameToResend.ToString() +
                DotSeparator.ToString() +
                FileNameExtension.ToString() +
                Filler1.ToString() +
                RejectCode.ToString() +
                Filler2.ToString() +
                RejectExplanation.ToString() +
                Filler3.ToString();
        }

        public static G50Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G50Detail rtn = new G50Detail();

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.FileNameToResend.Value = detailLine.Substring(rtn.FileNameToResend.Offset, rtn.FileNameToResend.Length).Trim();
            rtn.DotSeparator.Value = detailLine.Substring(rtn.DotSeparator.Offset, rtn.DotSeparator.Length).Trim();
            rtn.FileNameExtension.Value = detailLine.Substring(rtn.FileNameExtension.Offset, rtn.FileNameExtension.Length).Trim();
            rtn.RejectCode.Value = detailLine.Substring(rtn.RejectCode.Offset, rtn.RejectCode.Length).Trim();
            rtn.RejectExplanation.Value = detailLine.Substring(rtn.RejectExplanation.Offset, rtn.RejectExplanation.Length).Trim();

            return rtn;
        }

        #region IComparable<G50Detail> Members

        public int CompareTo(G50Detail other)
        {
            int pickupComp = this.FileNameToResend.Value.CompareTo(other.FileNameToResend.Value);
            int dropComp = this.RejectCode.Value.CompareTo(other.RejectCode.Value);

            if (pickupComp == 0 && dropComp == 0)
                return 0;
            else if (pickupComp < 0)
                return -1;
            else if (pickupComp > 0)
                return 1;
            else
            {
                if (dropComp < 0)
                    return -1;
                else if (dropComp > 0)
                    return 1;
                else return 0;
            }
        }

        #endregion
    }

    public class G50Detail_Comparer : IComparer<G50Detail>
    {
        #region IComparer<G50Detail> Members

        public int Compare(G50Detail x, G50Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G50Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G50Trailer()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = Utils.EOF, Required = true };
            TransmitRecordCount = new FixedPositionItem<int>() { Offset = 5, Length = 6, Value = 0, Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 11, Length = 339, Value = string.Empty, Required = false };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                TransmitRecordCount.ToString() +
                Filler.ToString();
        }

        public static G50Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G50Trailer rtn = new G50Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
