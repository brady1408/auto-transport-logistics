using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;
using System.Data.Linq;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Allocation Cancellation
    /// </summary>
    public class G78
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G78Detail> _detail = new List<G78Detail>();
        public G78Header Header { get; set; }
        public G78Trailer Trailer { get; set; }
        public List<G78Detail> Detail
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

        public G78()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G78Header();
            Trailer = new G78Trailer();
        }

        public static G78 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G78 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G78 rtn = new G78();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G78Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "CVT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G78Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G78";
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
                    case "CVT01":
                        G78Detail d = null;
                        try
                        {
                            d = G78Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Detail)";
                            err.Description = "Missing required information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G78";
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
                            rtn.Trailer = G78Trailer.Load(line);
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
                            err.EdiSet = "G78";
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
                        break;
                }
                break;
            }

            return rtn;
        }

        public bool Process(ATLDbDataContext db)
        {
            DateTime creation = CreatedDateTime;

            foreach (G78Detail det in Detail)
            {
                List<D10> d10s = null;
                try
                {
                    var q = (from p in db.D10s
                                where p.VIN.Trim().Equals(det.VIN.Value) &&
                                     p.AuthorizationCode.Trim().Equals(det.ExistingAllocationNumber.Value) &&
                                     !p.Status.Trim().Equals("Canceled")
                                orderby p.D10Id descending
                                select p);

                    if(q.Count() < 1)
                        continue;

                    d10s = q.ToList();
                }
                catch
                {
                    continue;
                }

                if (d10s == null) continue;

                foreach (D10 d10 in d10s)
                {
                    if (d10.SF1.Trim().Equals("")) continue;
                    DateTime transDT;
                    DateTime.TryParseExact(d10.SF1.Trim(), "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture, 
                        DateTimeStyles.None, out transDT);

                    if (transDT > det.CancellationDate.Value || transDT == DateTime.MinValue || transDT == DateTime.MaxValue)
                        continue;

                    d10.Status = "Canceled";
                    d10.UpdatedBy = "ACES";
                    d10.UpdatedTimeString = CreatedDateTime;

                    D11 d11 = new D11();
                    d11.D10Id = d10.D10Id;
                    d11.CreatedBy = "ACES";
                    d11.NoteDate = det.CancellationDate.Value;
                    d11.Description = string.Format("VIN Canceled by ACES on {0}; Status Code: {1}",
                        det.CancellationDate.Value,
                        det.StatusCodeNumber.Value);

                    db.D11s.InsertOnSubmit(d11);

                    try
                    {
                        db.SubmitChanges(ConflictMode.ContinueOnConflict);
                    }
                    catch (Exception ex) 
                    { 
                        AppLog.WriteExceptionToLog(ex, null, true);
                        return false;
                    }
                }
            }

            return true;
        }
    }

    public class G78Header
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

        public G78Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "CVT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G78", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Required = true };
            PortCode = new FixedPositionItem<string>() { Offset = 28, Length = 2, Value = string.Empty };
            CustomerCode = new FixedPositionItem<string>() { Offset = 30, Length = 10, Value = string.Empty, Required = true };
            TotalRecordCount = new FixedPositionItem<int>() { Offset = 40, Length = 6, Value = 0, Format = "{0:000000}", Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 46, Length = 204, Value = new string(Utils.FillerChar, 204) };
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

        public static G78Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G78Header rtn = new G78Header();

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

    public class G78Detail : IComparable<G78Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<DateTime> CancellationDate { get; set; }
        public FixedPositionItem<string> AllocationDealer { get; set; }
        public FixedPositionItem<string> ExistingAllocationNumber { get; set; }
        public FixedPositionItem<string> StatusCodeNumber { get; set; }
        public FixedPositionItem<string> Filler { get; set; }
        public bool DoInsert { get; set; }

        public G78Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "CVT01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            CancellationDate = new FixedPositionItem<DateTime>() { Offset = 22, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            AllocationDealer = new FixedPositionItem<string>() { Offset = 30, Length = 6, Value = string.Empty, Required = true };
            ExistingAllocationNumber = new FixedPositionItem<string>() { Offset = 36, Length = 12, Value = string.Empty, Required = true };
            StatusCodeNumber = new FixedPositionItem<string>() { Offset = 48, Length = 3, Value = string.Empty };
            Filler = new FixedPositionItem<string>() { Offset = 51, Length = 199, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                CancellationDate.ToString() +
                AllocationDealer.ToString() +
                ExistingAllocationNumber.ToString() +
                StatusCodeNumber.ToString() +
                Filler.ToString();
        }

        public static G78Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G78Detail rtn = new G78Detail();

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();

            string cdt = detailLine.Substring(rtn.CancellationDate.Offset, rtn.CancellationDate.Length).Trim();
            DateTime temp;
            DateTime.TryParseExact(cdt, "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.CancellationDate.Value = temp;

            rtn.AllocationDealer.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.ExistingAllocationNumber.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.StatusCodeNumber.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();

            return rtn;
        }

        #region IComparable<G78Detail> Members

        public int CompareTo(G78Detail other)
        {
            int pickupComp = this.VIN.Value.CompareTo(other.VIN.Value);
            int dropComp = this.ExistingAllocationNumber.Value.CompareTo(other.ExistingAllocationNumber.Value);

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

    public class G78Detail_Comparer : IComparer<G78Detail>
    {
        #region IComparer<G78Detail> Members

        public int Compare(G78Detail x, G78Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G78Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G78Trailer()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = Utils.EOF, Required = true };
            TransmitRecordCount = new FixedPositionItem<int>() { Offset = 5, Length = 6, Value = 0, Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 11, Length = 239, Value = new string(Utils.FillerChar, 239), Required = false };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                TransmitRecordCount.ToString() +
                Filler.ToString();
        }

        public static G78Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G78Trailer rtn = new G78Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
