using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Payment
    /// </summary>
    public class G96
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G96Detail> _detail = new List<G96Detail>();
        public G96Header Header { get; set; }
        public G96Trailer Trailer { get; set; }
        public List<G96Detail> Detail
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

        public G96()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G96Header();
            Trailer = new G96Trailer();
        }

        public static G96 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G96 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G96 rtn = new G96();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G96Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "PYT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G96Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G96";
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
                            return null;
                        }
                        continue;
                    case "PYT01":
                        G96Detail d = null;
                        try
                        {
                            d = G96Detail.Load(line);
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
                            rtn.Trailer = G96Trailer.Load(line);
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
                            err.EdiSet = "G96";
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

            List<ACES_G96> newG96s = new List<ACES_G96>(Detail.Count);
            foreach (G96Detail det in Detail)
            {
                int checkCount = 0;

                try
                {
                    checkCount = (from p in db.ACES_G96s
                                    where p.check_amt == det.CheckAmount.Value &&
                                    p.check_date_time == det.CheckDate.Value &&
                                    p.check_num.Trim().Equals(det.CheckNumber.Value) &&
                                    p.glovis_voucher_num.Trim().Equals(det.GLOVISVoucherNumber.Value) &&
                                    p.payee_dir_num.Trim().Equals(det.PayeeVendorNumber.Value) &&
                                    p.voucher_amt == det.VoucherAmount.Value
                                    select p).Count();
                }
                catch { }

                if (checkCount < 1)
                {
                    ACES_G96 g96 = new ACES_G96();
                    g96.check_amt = det.CheckAmount.Value;
                    g96.check_date_time = det.CheckDate.Value;
                    g96.check_num = det.CheckNumber.Value;
                    g96.glovis_voucher_num = det.GLOVISVoucherNumber.Value;
                    g96.payee_dir_num = det.PayeeVendorNumber.Value;
                    g96.voucher_amt = det.VoucherAmount.Value;

                    newG96s.Add(g96);
                }
            }

            try
            {
                ACES_G96 dbG96 = ((from p in db.ACES_G96s
                                   orderby p.ACES_G96Id descending
                                   select p).FirstOrDefault() as ACES_G96);

                int startId = 1;
                if (dbG96 != null)
                    startId = dbG96.ACES_G96Id + 1;

                foreach (ACES_G96 g in newG96s)
                {
                    g.ACES_G96Id = startId++;
                }

            }
            catch (InvalidOperationException ioex)
            {
                if (ioex.Message.Equals("Sequence contains no elements"))
                {
                    int newIds = 1;
                    foreach (ACES_G96 g in newG96s)
                    {
                        g.ACES_G96Id = newIds++;
                    }
                }
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }

            try
            {
                db.ACES_G96s.InsertAllOnSubmit(newG96s);
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

    public class G96Header
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

        public G96Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "PYT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G96", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Required = true };
            PortCode = new FixedPositionItem<string>() { Offset = 28, Length = 2, Value = string.Empty };
            CustomerCode = new FixedPositionItem<string>() { Offset = 30, Length = 10, Value = string.Empty, Required = false };
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

        public static G96Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G96Header rtn = new G96Header();

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

    public class G96Detail : IComparable<G96Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> GLOVISVoucherNumber { get; set; }
        public FixedPositionItem<string> PayeeVendorNumber { get; set; }
        public FixedPositionItem<DateTime> CheckDate { get; set; }
        public FixedPositionItem<string> CheckNumber { get; set; }
        public FixedPositionItem<decimal> CheckAmount { get; set; }
        public FixedPositionItem<decimal> VoucherAmount { get; set; }
        public FixedPositionItem<string> Filler { get; private set; }
        public bool DoInsert { get; set; }

        public G96Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "PYT01", Required = true };
            GLOVISVoucherNumber = new FixedPositionItem<string>() { Offset = 5, Length = 10, Value = string.Empty, Required = true };
            PayeeVendorNumber = new FixedPositionItem<string>() { Offset = 15, Length = 5, Value = string.Empty, Required = true };
            CheckDate = new FixedPositionItem<DateTime>() { Offset = 20, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            CheckNumber = new FixedPositionItem<string>() { Offset = 28, Length = 10, Value = string.Empty, Required = true };
            CheckAmount = new FixedPositionItem<decimal>() { Offset = 38, Length = 9, Value = 0M, Format = "{0:000000000}", Required = true };
            VoucherAmount = new FixedPositionItem<decimal>() { Offset = 47, Length = 9, Value = 0M, Format = "{0:000000000}", Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 56, Length = 194, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                GLOVISVoucherNumber.ToString() +
                PayeeVendorNumber.ToString() +
                CheckDate.ToString() +
                CheckNumber.ToString() +
                CheckAmount.ToString() +
                VoucherAmount.ToString() +
                Filler.ToString();
        }

        public static G96Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G96Detail rtn = new G96Detail();
            DateTime chDT;
            decimal amt;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.GLOVISVoucherNumber.Value = detailLine.Substring(rtn.GLOVISVoucherNumber.Offset, 
                rtn.GLOVISVoucherNumber.Length).Trim();
            rtn.PayeeVendorNumber.Value = detailLine.Substring(rtn.PayeeVendorNumber.Offset, rtn.PayeeVendorNumber.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.CheckDate.Offset, rtn.CheckDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out chDT);
            rtn.CheckDate.Value = chDT;

            rtn.CheckNumber.Value = detailLine.Substring(rtn.CheckNumber.Offset, rtn.CheckNumber.Length).Trim();

            decimal.TryParse(detailLine.Substring(rtn.CheckAmount.Offset, rtn.CheckAmount.Length).Trim(), out amt);
            rtn.CheckAmount.Value = amt / 100;

            decimal.TryParse(detailLine.Substring(rtn.VoucherAmount.Offset, rtn.VoucherAmount.Length).Trim(), out amt);
            rtn.VoucherAmount.Value = amt / 100;

            return rtn;
        }

        #region IComparable<G96Detail> Members

        public int CompareTo(G96Detail other)
        {
            int pickupComp = this.GLOVISVoucherNumber.Value.CompareTo(other.GLOVISVoucherNumber.Value);
            int dropComp = this.CheckNumber.Value.CompareTo(other.CheckNumber.Value);

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

    public class G96Detail_Comparer : IComparer<G96Detail>
    {
        #region IComparer<G96Detail> Members

        public int Compare(G96Detail x, G96Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G96Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G96Trailer()
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

        public static G96Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G96Trailer rtn = new G96Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
