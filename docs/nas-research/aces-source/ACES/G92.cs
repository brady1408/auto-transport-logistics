using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Invoice Audit Report
    /// </summary>
    public class G92
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G92Detail> _detail = new List<G92Detail>();
        public G92Header Header { get; set; }
        public G92Trailer Trailer { get; set; }
        public List<G92Detail> Detail
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

        public G92()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G92Header();
            Trailer = new G92Trailer();
        }

        public static G92 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G92 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G92 rtn = new G92();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G92Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "IAT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G92Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
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
                        }
                        continue;
                    case "IAT01":
                        G92Detail d = null;
                        try
                        {
                            d = G92Detail.Load(line);
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
                            rtn.Trailer = G92Trailer.Load(line);
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

            //Detail.Sort(new G92Detail_Comparer());

            List<ACES_G92> vouchers = new List<ACES_G92>(Detail.Count);
            foreach (G92Detail det in Detail)
            {
                int voucherCount = 0;

                try
                {
                    voucherCount = (from p in db.ACES_G92s
                                    where p.VoucherNumber == det.VoucherNumber.Value &&
                                    p.VIN.Trim().Equals(det.VIN.Value) &&
                                    p.InvoiceNumber.Trim().Equals(det.InvoiceNumber.Value) &&
                                    p.InvoiceDate == det.InvoiceDate.Value &&
                                    p.ApprovedAmt == det.AmountApproved.Value &&
                                    p.DisapprovedAmt == det.AmountDisapproved.Value &&
                                    p.LastStatusCode.Trim().Equals(det.LastStatusCode.Value)
                                    select p).Count();
                }
                catch { }

                if (voucherCount < 1)
                {
                    ACES_G92 voucher = new ACES_G92();
                    voucher.InvoiceNumber = det.InvoiceNumber.Value;
                    voucher.InvoiceDate = det.InvoiceDate.Value;
                    voucher.VIN = det.VIN.Value;
                    voucher.AccountNumber = det.AccountNumber.Value;
                    voucher.DamageCode = det.DamageCode.Value;
                    voucher.InvoiceAmt = det.InvoiceAmount.Value;
                    voucher.AuditReportCode = det.AuditReportCode.Value;
                    voucher.VoucherNumber = det.VoucherNumber.Value;
                    voucher.ApprovedAmt = det.AmountApproved.Value;
                    voucher.DisapprovedAmt = det.AmountDisapproved.Value;
                    voucher.AllocationDealer = det.AllocationDealer.Value;
                    voucher.LastStatusCode = det.LastStatusCode.Value;
                    voucher.CreatedTimeString = creation;
                    voucher.EDIFileName = TransmissionInfo.LocalFile.Name;

                    try
                    {
                        D10 d10 = (from p in db.D10s
                                   where p.VIN == det.VIN.Value
                                   select p).FirstOrDefault() as D10;

                        d10.RemittanceAmt = det.AmountApproved.Value;
                        d10.UpdatedTimeString = creation;
                    }
                    catch (Exception ex) { }

                    vouchers.Add(voucher);
                }
            }

            try
            {
                ACES_G92 cs = (from p in db.ACES_G92s
                               orderby p.ACES_G92Id descending
                               select p).FirstOrDefault() as ACES_G92;

                int startId = 1;
                if (cs != null)
                    startId = cs.ACES_G92Id + 1;

                foreach (ACES_G92 g in vouchers)
                {
                    g.ACES_G92Id = startId++;
                }
            }
            catch (InvalidOperationException ioex)
            {
                if (ioex.Message.Equals("Sequence contains no elements"))
                {
                    int newIds = 1;
                    foreach (ACES_G92 g in vouchers)
                    {
                        g.ACES_G92Id = newIds++;
                    }
                }
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }

            try
            {
                db.ACES_G92s.InsertAllOnSubmit(vouchers);
                db.SubmitChanges(System.Data.Linq.ConflictMode.FailOnFirstConflict);
            }
            catch (Exception ex)
            {
                db.Log.Flush();
                AppLog.WriteExceptionToLog(ex, null, true);
                return false;
            }

            return true;
        }
    }

    public class G92Header
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

        public G92Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "IAT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G92", Required = true };
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

        public static G92Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G92Header rtn = new G92Header();

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

    public class G92Detail : IComparable<G92Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> InvoiceNumber { get; set; }
        public FixedPositionItem<string> Filler1 { get; private set; }
        public FixedPositionItem<DateTime> InvoiceDate { get; set; }
        public FixedPositionItem<string> Filler2 { get; private set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> Filler3 { get; private set; }
        public FixedPositionItem<string> AccountNumber { get; set; }
        public FixedPositionItem<string> Filler4 { get; private set; }
        public FixedPositionItem<string> DamageCode { get; set; }
        public FixedPositionItem<string> Filler5 { get; private set; }
        public FixedPositionItem<string> WorkOrderCode { get; set; }
        public FixedPositionItem<string> Filler6 { get; private set; }
        public FixedPositionItem<decimal> InvoiceAmount { get; set; }
        public FixedPositionItem<string> Filler7 { get; private set; }
        public FixedPositionItem<string> AuditReportCode { get; set; }
        public FixedPositionItem<string> Filler8 { get; private set; }
        public FixedPositionItem<int> VoucherNumber { get; set; }
        public FixedPositionItem<string> Filler9 { get; private set; }
        public FixedPositionItem<decimal> AmountApproved { get; set; }
        public FixedPositionItem<string> Filler10 { get; private set; }
        public FixedPositionItem<decimal> AmountDisapproved { get; set; }
        public FixedPositionItem<string> Filler11 { get; private set; }
        public FixedPositionItem<decimal> ManufacturerStandardContractRate { get; set; }
        public FixedPositionItem<string> Filler12 { get; private set; }
        public FixedPositionItem<string> AllocationDealer { get; set; }
        public FixedPositionItem<string> LastStatusCode { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<string> Filler13 { get; private set; }
        public FixedPositionItem<string> AuditCodeDescription { get; set; }
        public FixedPositionItem<string> Filler14 { get; private set; }
        public bool DoInsert { get; set; }

        public G92Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "IAT01", Required = true };
            InvoiceNumber = new FixedPositionItem<string>() { Offset = 5, Length = 15, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 20, Length = 1, Value = string.Empty };
            InvoiceDate = new FixedPositionItem<DateTime>() { Offset = 21, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            Filler2 = new FixedPositionItem<string>() { Offset = 29, Length = 1, Value = string.Empty };
            VIN = new FixedPositionItem<string>() { Offset = 30, Length = 17, Value = string.Empty, Required = true };
            Filler3 = new FixedPositionItem<string>() { Offset = 47, Length = 1, Value = string.Empty };
            AccountNumber = new FixedPositionItem<string>() { Offset = 48, Length = 4, Value = string.Empty, Required = true };
            Filler4 = new FixedPositionItem<string>() { Offset = 52, Length = 1, Value = string.Empty };
            DamageCode = new FixedPositionItem<string>() { Offset = 53, Length = 6, Value = string.Empty };
            Filler5 = new FixedPositionItem<string>() { Offset = 59, Length = 1, Value = string.Empty };
            WorkOrderCode = new FixedPositionItem<string>() { Offset = 60, Length = 4, Value = string.Empty };
            Filler6 = new FixedPositionItem<string>() { Offset = 64, Length = 1, Value = string.Empty };
            InvoiceAmount = new FixedPositionItem<decimal>() { Offset = 65, Length = 8, Value = 0M, Format = "{0:00000000}", Required = true };
            Filler7 = new FixedPositionItem<string>() { Offset = 73, Length = 1, Value = string.Empty };
            AuditReportCode = new FixedPositionItem<string>() { Offset = 74, Length = 4, Value = string.Empty, Required = true };
            Filler8 = new FixedPositionItem<string>() { Offset = 78, Length = 1, Value = string.Empty };
            VoucherNumber = new FixedPositionItem<int>() { Offset = 79, Length = 8, Value = 0, Format = "{0:00000000}"};
            Filler9 = new FixedPositionItem<string>() { Offset = 87, Length = 1, Value = string.Empty };
            AmountApproved = new FixedPositionItem<decimal>() { Offset = 88, Length = 8, Value = 0M, Format = "{0:00000000}" };
            Filler10 = new FixedPositionItem<string>() { Offset = 96, Length = 1, Value = string.Empty };
            AmountDisapproved = new FixedPositionItem<decimal>() { Offset = 97, Length = 8, Value = 0M, Format = "{0:00000000}" };
            Filler11 = new FixedPositionItem<string>() { Offset = 105, Length = 1, Value = string.Empty };
            ManufacturerStandardContractRate = new FixedPositionItem<decimal>() { Offset = 106, Length = 8, Value = 0M, Format = "{0:00000000}" };
            Filler12 = new FixedPositionItem<string>() { Offset = 114, Length = 1, Value = string.Empty };
            AllocationDealer = new FixedPositionItem<string>() { Offset = 115, Length = 7, Value = string.Empty };
            LastStatusCode = new FixedPositionItem<string>() { Offset = 122, Length = 3, Value = string.Empty };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 125, Length = 12, Value = string.Empty, Required = true };
            Filler13 = new FixedPositionItem<string>() { Offset = 137, Length = 1, Value = string.Empty };
            AuditCodeDescription = new FixedPositionItem<string>() { Offset = 138, Length = 75, Value = string.Empty };
            Filler14 = new FixedPositionItem<string>() { Offset = 213, Length = 37, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                InvoiceNumber.ToString() +
                Filler1.ToString() +
                InvoiceDate.ToString() +
                Filler2.ToString() +
                VIN.ToString() +
                Filler3.ToString() +
                AccountNumber.ToString() +
                Filler4.ToString() +
                DamageCode.ToString() +
                Filler5.ToString() +
                WorkOrderCode.ToString() +
                Filler6.ToString() +
                InvoiceAmount.ToString() +
                Filler7.ToString() +
                AuditReportCode.ToString() +
                Filler8.ToString() +
                VoucherNumber.ToString() +
                Filler9.ToString() +
                AmountApproved.ToString() +
                Filler10.ToString() +
                AmountDisapproved.ToString() +
                Filler11.ToString() +
                ManufacturerStandardContractRate.ToString() +
                Filler12.ToString() +
                AllocationDealer.ToString() +
                LastStatusCode.ToString() +
                ShipmentAuthorizationCode.ToString() +
                Filler13.ToString() +
                AuditCodeDescription.ToString() +
                Filler14.ToString();
        }

        public static G92Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G92Detail rtn = new G92Detail();

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.InvoiceNumber.Value = detailLine.Substring(rtn.InvoiceNumber.Offset, rtn.InvoiceNumber.Length).Trim();
            string invoiceDateTimeString = detailLine.Substring(rtn.InvoiceDate.Offset, rtn.InvoiceDate.Length).Trim();
            DateTime invDT;
            DateTime.TryParseExact(invoiceDateTimeString, "yyyyMMdd", CultureInfo.InvariantCulture, 
                DateTimeStyles.NoCurrentDateDefault, out invDT);
            rtn.InvoiceDate.Value = invDT;
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.AccountNumber.Value = detailLine.Substring(rtn.AccountNumber.Offset, rtn.AccountNumber.Length).Trim();
            rtn.DamageCode.Value = detailLine.Substring(rtn.DamageCode.Offset, rtn.DamageCode.Length).Trim();
            rtn.WorkOrderCode.Value = detailLine.Substring(rtn.WorkOrderCode.Offset, rtn.WorkOrderCode.Length).Trim();
            decimal invAmt;
            decimal.TryParse(detailLine.Substring(rtn.InvoiceAmount.Offset, rtn.InvoiceAmount.Length).Trim(), out invAmt);
            rtn.InvoiceAmount.Value = invAmt / 100;
            rtn.AuditReportCode.Value = detailLine.Substring(rtn.AuditReportCode.Offset, rtn.AuditReportCode.Length).Trim();
            int vn;
            int.TryParse(detailLine.Substring(rtn.VoucherNumber.Offset, rtn.VoucherNumber.Length).Trim(), out vn);
            rtn.VoucherNumber.Value = vn;
            decimal appAmt;
            decimal.TryParse(detailLine.Substring(rtn.AmountApproved.Offset, rtn.AmountApproved.Length).Trim(), out appAmt);
            rtn.AmountApproved.Value = appAmt / 100;
            decimal disAmt;
            decimal.TryParse(detailLine.Substring(rtn.AmountDisapproved.Offset, rtn.AmountDisapproved.Length).Trim(), out disAmt);
            rtn.AmountDisapproved.Value = disAmt / 100;
            decimal stdRate;
            decimal.TryParse(detailLine.Substring(rtn.ManufacturerStandardContractRate.Offset, rtn.ManufacturerStandardContractRate.Length).Trim(), out stdRate);
            rtn.ManufacturerStandardContractRate.Value = stdRate / 100;
            rtn.AllocationDealer.Value = detailLine.Substring(rtn.AllocationDealer.Offset, rtn.AllocationDealer.Length).Trim();
            rtn.LastStatusCode.Value = detailLine.Substring(rtn.LastStatusCode.Offset, rtn.LastStatusCode.Length).Trim();
            rtn.ShipmentAuthorizationCode.Value = detailLine.Substring(rtn.ShipmentAuthorizationCode.Offset, rtn.ShipmentAuthorizationCode.Length).Trim();
            rtn.AuditCodeDescription.Value = detailLine.Substring(rtn.AuditCodeDescription.Offset, rtn.AuditCodeDescription.Length).Trim();
            
            return rtn;
        }

        #region IComparable<G92Detail> Members

        public int CompareTo(G92Detail other)
        {
            int invNumComp = this.InvoiceNumber.Value.CompareTo(other.InvoiceNumber.Value);
            int vinComp = this.VIN.Value.CompareTo(other.VIN.Value);

            if (invNumComp == 0 && vinComp == 0)
                return 0;
            else if (invNumComp < 0)
                return -1;
            else if (invNumComp > 0)
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

    public class G92Detail_Comparer : IComparer<G92Detail>
    {
        #region IComparer<G92Detail> Members

        public int Compare(G92Detail x, G92Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G92Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G92Trailer()
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

        public static G92Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G92Trailer rtn = new G92Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
