using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Data Validation Error
    /// </summary>
    public class G51
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G51Detail> _detail = new List<G51Detail>();
        public G51Header Header { get; set; }
        public G51Trailer Trailer { get; set; }
        public List<G51Detail> Detail
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

        public G51()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G51Header();
            Trailer = new G51Trailer();
        }

        public static G51 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G51 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G51 rtn = new G51();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G51Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "DVP00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G51Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G51";
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
                    case "DVP01":
                        G51Detail d = null;
                        try
                        {
                            d = G51Detail.Load(line);
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
                            rtn.Trailer = G51Trailer.Load(line);
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
                            err.EdiSet = "G51";
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

            List<ACES_G51> newG51s = new List<ACES_G51>(Detail.Count);
            foreach (G51Detail det in Detail)
            {
                int errCount = 0;
                try
                {
                    errCount = (from p in db.ACES_G51s
                                where p.FileName.Trim().Equals(det.FileName.Value) &&
                                p.ACES_ErrorCode.Trim().Equals(det.ErrorCode.Value) &&
                                p.ACES_ErrorCodeDesc.Trim().Equals(det.ErrorCodeDescription.Value) &&
                                p.FileLine == det.FileLineNumber.Value &&
                                p.FilePosition == det.FilePositionNumber.Value &&
                                p.SortCode.Trim().Equals(det.SortCode.Value) &&
                                p.VIN.Trim().Equals(det.VIN.Value)
                                select p).Count();
                }
                catch { }

                ACES_G51 g51 = new ACES_G51();
                g51.ACES_ErrorCode = det.ErrorCode.Value.Substring(0,3);
                g51.ACES_ErrorCodeDesc = det.ErrorCodeDescription.Value.Length > 36 ?
                    det.ErrorCodeDescription.Value.Substring(0, 37) : det.ErrorCodeDescription.Value;
                g51.FileLine = det.FileLineNumber.Value;
                g51.FileName = det.FileName.Value.Length > 9 ? det.FileName.Value.Substring(0, 10) : det.FileName.Value;
                g51.FilePosition = det.FilePositionNumber.Value;
                g51.SortCode = det.SortCode.Value.Length > 2 ? det.SortCode.Value.Substring(0, 3) : det.SortCode.Value;
                g51.VIN = det.VIN.Value.Length > 16 ? det.VIN.Value.Substring(0, 17) : det.VIN.Value;

                newG51s.Add(g51);
            }

            bool exc = false;
            try
            {
                ACES_G51 dbG51 = ((from p in db.ACES_G51s
                                   orderby p.ACES_G51Id descending
                                   select p).FirstOrDefault() as ACES_G51);

                int startId = 1;
                if (dbG51 != null)
                    startId = dbG51.ACES_G51Id + 1;

                foreach (ACES_G51 g in newG51s)
                {
                    g.ACES_G51Id = startId++;
                }
            }
            catch (InvalidOperationException ioex)
            {
                if (ioex.Message.Equals("Sequence contains no elements"))
                {
                    int newIds = 1;
                    foreach (ACES_G51 g in newG51s)
                    {
                        g.ACES_G51Id = newIds++;
                    }
                }
                exc = true;
            }
            catch (Exception ex) 
            { 
                AppLog.WriteExceptionToLog(ex, null, true);
                exc = true;
            }
            finally
            {
                db.ACES_G51s.InsertAllOnSubmit(newG51s);
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
            }

            return !exc;
        }
    }

    public class G51Header
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

        public G51Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "DVP00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G51", Required = true };
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

        public static G51Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G51Header rtn = new G51Header();

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

    public class G51Detail : IComparable<G51Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> FileName { get; set; }
        public FixedPositionItem<string> Filler1 { get; set; }
        public FixedPositionItem<string> SortCode { get; set; }
        public FixedPositionItem<string> Filler2 { get; set; }
        public FixedPositionItem<int> FileLineNumber { get; set; }
        public FixedPositionItem<string> Filler3 { get; set; }
        public FixedPositionItem<int> FilePositionNumber { get; set; }
        public FixedPositionItem<string> Filler4 { get; set; }
        public FixedPositionItem<string> ErrorCode { get; set; }
        public FixedPositionItem<string> Filler5 { get; set; }
        public FixedPositionItem<string> ErrorCodeDescription { get; set; }
        public FixedPositionItem<string> Filler6 { get; set; }
        public FixedPositionItem<string> ReferenceNumber1 { get; set; }
        public FixedPositionItem<string> ReferenceNumber2 { get; set; }
        public FixedPositionItem<string> Filler7 { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> Filler8 { get; set; }
        public bool DoInsert { get; set; }

        public G51Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "DVP01", Required = true };
            FileName = new FixedPositionItem<string>() { Offset = 5, Length = 10, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 15, Length = 2, Value = string.Empty };
            SortCode = new FixedPositionItem<string>() { Offset = 17, Length = 3, Value = string.Empty, Required = true };
            Filler2 = new FixedPositionItem<string>() { Offset = 20, Length = 2, Value = string.Empty };
            FileLineNumber = new FixedPositionItem<int>() { Offset = 22, Length = 4, Format = "{0:0000}", Required = true };
            Filler3 = new FixedPositionItem<string>() { Offset = 26, Length = 2, Value = string.Empty };
            FilePositionNumber = new FixedPositionItem<int>() { Offset = 28, Length = 3, Format = "{0:000}", Required = true };
            Filler4 = new FixedPositionItem<string>() { Offset = 31, Length = 2, Value = string.Empty };
            ErrorCode = new FixedPositionItem<string>() { Offset = 33, Length = 8, Value = string.Empty, Required = true };
            Filler5 = new FixedPositionItem<string>() { Offset = 41, Length = 2, Value = string.Empty };
            ErrorCodeDescription = new FixedPositionItem<string>() { Offset = 43, Length = 37, Value = string.Empty, Required = true };
            Filler6 = new FixedPositionItem<string>() { Offset = 80, Length = 2, Value = string.Empty };
            ReferenceNumber1 = new FixedPositionItem<string>() { Offset = 82, Length = 10, Value = string.Empty, Required = true };
            ReferenceNumber2 = new FixedPositionItem<string>() { Offset = 92, Length = 10, Value = string.Empty, Required = true };
            Filler7 = new FixedPositionItem<string>() { Offset = 102, Length = 25, Value = string.Empty };
            VIN = new FixedPositionItem<string>() { Offset = 127, Length = 17, Value = string.Empty, Required = true };
            Filler8 = new FixedPositionItem<string>() { Offset = 144, Length = 206, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                FileName.ToString() +
                Filler1.ToString() +
                SortCode.ToString() +
                Filler2.ToString() +
                FileLineNumber.ToString() +
                Filler3.ToString() +
                FilePositionNumber.ToString() +
                Filler4.ToString() +
                ErrorCode.ToString() +
                Filler5.ToString() +
                ErrorCodeDescription.ToString() +
                Filler6.ToString() +
                ReferenceNumber1.ToString() +
                ReferenceNumber2.ToString() +
                Filler7.ToString() +
                VIN.ToString() +
                Filler8.ToString();
        }

        public static G51Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G51Detail rtn = new G51Detail();
            int temp;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.FileName.Value = detailLine.Substring(rtn.FileName.Offset, rtn.FileName.Length).Trim();
            rtn.SortCode.Value = detailLine.Substring(rtn.SortCode.Offset, rtn.SortCode.Length).Trim();

            int.TryParse(detailLine.Substring(rtn.FileLineNumber.Offset, rtn.FileLineNumber.Length).Trim(), out temp);
            rtn.FileLineNumber.Value = temp;

            int.TryParse(detailLine.Substring(rtn.FilePositionNumber.Offset, rtn.FilePositionNumber.Length).Trim(), out temp);
            rtn.FilePositionNumber.Value = temp;

            rtn.ErrorCode.Value = detailLine.Substring(rtn.ErrorCode.Offset, rtn.ErrorCode.Length).Trim();
            rtn.ErrorCodeDescription.Value = detailLine.Substring(rtn.ErrorCodeDescription.Offset,
                rtn.ErrorCodeDescription.Length).Trim();
            rtn.ReferenceNumber1.Value = detailLine.Substring(rtn.ReferenceNumber1.Offset, rtn.ReferenceNumber1.Length).Trim();
            rtn.ReferenceNumber2.Value = detailLine.Substring(rtn.ReferenceNumber2.Offset, rtn.ReferenceNumber2.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();

            return rtn;
        }

        #region IComparable<G51Detail> Members

        public int CompareTo(G51Detail other)
        {
            int fileComp = this.FileName.Value.CompareTo(other.FileName.Value);
            int vinComp = this.VIN.Value.CompareTo(other.VIN.Value);

            if (fileComp == 0 && vinComp == 0)
                return 0;
            else if (fileComp < 0)
                return -1;
            else if (fileComp > 0)
                return 1;
            else
            {
                if (vinComp < 0)
                    return -1;
                else if (vinComp > 0)
                    return 1;
                else return 0;
            }
        }

        #endregion
    }

    public class G51Detail_Comparer : IComparer<G51Detail>
    {
        #region IComparer<G51Detail> Members

        public int Compare(G51Detail x, G51Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G51Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G51Trailer()
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

        public static G51Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G51Trailer rtn = new G51Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
