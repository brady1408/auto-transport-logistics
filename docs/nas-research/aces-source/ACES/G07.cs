using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Release of Truck Shipment
    /// </summary>
    public class G07
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G07Detail> _detail = new List<G07Detail>();
        public G07Header Header { get; set; }
        public G07Trailer Trailer { get; set; }
        public List<G07Detail> Detail
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

        public G07()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G07Header();
            Trailer = new G07Trailer();
        }

        public static G07 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G07 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G07 rtn = new G07();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G07Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "RTT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G07Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G07";
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
                    case "RTT01":
                        G07Detail d = null;
                        try
                        {
                            d = G07Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Detail)";
                            err.Description = "Missing required information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G07";
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
                            rtn.Trailer = G07Trailer.Load(line);
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
                            err.EdiSet = "G07";
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

            Detail.Sort(new G07Detail_Comparer());

            foreach (G07Detail det in Detail)
            {
                D10 d10 = null;
                try
                {
                    d10 = (from p in db.D10s
                           where p.VIN.Equals(det.VIN.Value) &&
                                p.AuthorizationCode.Trim().Equals(det.AllocationNumber.Value) &&
                                p.Status.Trim().Equals("Inbound")
                           orderby p.D10Id descending
                           select p).FirstOrDefault() as D10;
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }
                if (d10 == null) continue;

                d10.BayTimeString = new DateTime(det.TenderDate.Value.Year, det.TenderDate.Value.Month, det.TenderDate.Value.Day,
                    det.TenderDate.Value.Hour, det.TenderDate.Value.Minute, det.TenderDate.Value.Second, det.TenderDate.Value.Millisecond, 
                    DateTimeKind.Unspecified);

                d10.UnitID = det.BayLocation.Value;
                d10.Color = det.ExteriorColor.Value;
                d10.Status = "Waiting";
                d10.UpdatedTimeString = DateTime.Now;
                d10.UpdatedBy = "ACES";

                if (det.DropShipFlag.Value)
                {
                    D00 d00 = (from p in db.D00s
                               where p.D00Id == d10.D00Id
                               select p).FirstOrDefault() as D00;

                    if (d00 != null)
                    {
                        d00.DropAddr1 = det.ShipToAddress1.Value.Length > 30 ? 
                            det.ShipToAddress1.Value.Substring(0,30) : det.ShipToAddress1.Value;
                        d00.DropAddr2 = det.ShipToAddress2.Value.Length > 30 ? 
                            det.ShipToAddress2.Value.Substring(0,30) : det.ShipToAddress2.Value;
                        string[] cityState = det.ShipToAddress3.Value.Split(new char[] { ',' }, StringSplitOptions.RemoveEmptyEntries);
                        if (cityState.Length > 0)
                            d00.DropCity = cityState[0];
                        if (cityState.Length > 1)
                            d00.DropState = cityState[1].Trim();

                        d00.DropZip = det.ZipCode.Value;
                        d00.DropContact = det.ContactName.Value.Length > 30 ? 
                            det.ContactName.Value.Substring(0,30) : det.ContactName.Value;
                        d00.DropPhone = det.PhoneNumber.Value;
                        d00.DropShip = (byte)1;
                    }
                }

                try
                {
                    db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                }
                catch (Exception ex) 
                { 
                    AppLog.WriteExceptionToLog(ex, null, true);
                    return false;
                }
            }

            return true;
        }
    }

    public class G07Header
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

        public G07Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "RTT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G07", Required = true };
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

        public static G07Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G07Header rtn = new G07Header();

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

    public class G07Detail : IComparable<G07Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> AllocationNumber { get; set; }
        public FixedPositionItem<DateTime> TenderDate { get; set; }
        public FixedPositionItem<string> OriginCode { get; set; }
        public FixedPositionItem<string> BayLocation { get; set; }
        public FixedPositionItem<string> TruckDestinationDealerCode { get; set; }
        public FixedPositionItem<string> ShipToAddress1 { get; set; }
        public FixedPositionItem<string> ShipToAddress2 { get; set; }
        public FixedPositionItem<string> ShipToAddress3 { get; set; }
        public FixedPositionItem<string> ZipCode { get; set; }
        public FixedPositionItem<string> PhoneNumber { get; set; }
        public FixedPositionItem<string> ContactName { get; set; }
        public FixedPositionItem<bool> DropShipFlag { get; set; }
        public FixedPositionItem<DateTime> RequiredDeliveryDate { get; set; }
        public FixedPositionItem<string> ExteriorColor { get; set; }
        public FixedPositionItem<DateTime> TenderTime { get; set; }
        public FixedPositionItem<string> Filler { get; set; }
        public bool DoInsert { get; set; }

        public G07Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "RTT01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            AllocationNumber = new FixedPositionItem<string>() { Offset = 22, Length = 12, Value = string.Empty, Required = true };
            TenderDate = new FixedPositionItem<DateTime>() { Offset = 34, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            OriginCode = new FixedPositionItem<string>() { Offset = 42, Length = 7, Value = string.Empty, Required = true };
            BayLocation = new FixedPositionItem<string>() { Offset = 49, Length = 10, Value = string.Empty, Required = true };
            TruckDestinationDealerCode = new FixedPositionItem<string>() { Offset = 59, Length = 7, Value = string.Empty, Required = true };
            ShipToAddress1 = new FixedPositionItem<string>() { Offset = 66, Length = 30, Value = string.Empty };
            ShipToAddress2 = new FixedPositionItem<string>() { Offset = 96, Length = 30, Value = string.Empty };
            ShipToAddress3 = new FixedPositionItem<string>() { Offset = 126, Length = 30, Value = string.Empty };
            ZipCode = new FixedPositionItem<string>() { Offset = 156, Length = 10, Value = string.Empty };
            PhoneNumber = new FixedPositionItem<string>() { Offset = 166, Length = 20, Value = string.Empty };
            ContactName = new FixedPositionItem<string>() { Offset = 186, Length = 30, Value = string.Empty };
            DropShipFlag = new FixedPositionItem<bool>() { Offset = 216, Length = 1, Value = false, Format = "{0:Y;;N}", Required = true };
            RequiredDeliveryDate = new FixedPositionItem<DateTime>() { Offset = 217, Length = 8, Value = DateTime.Now.AddDays(3), Format = "{0:yyyyMMdd}", Required = true };
            ExteriorColor = new FixedPositionItem<string>() { Offset = 225, Length = 3, Value = string.Empty, Required = true };
            TenderTime = new FixedPositionItem<DateTime>() { Offset = 228, Length = 6, Value = DateTime.Now, Format = "{0:HHmmss}" };
            Filler = new FixedPositionItem<string>() { Offset = 234, Length = 16, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                AllocationNumber.ToString() +
                TenderDate.ToString() +
                OriginCode.ToString() +
                BayLocation.ToString() +
                TruckDestinationDealerCode.ToString() +
                ShipToAddress1.ToString() +
                ShipToAddress2.ToString() +
                ShipToAddress3.ToString() +
                ZipCode.ToString() +
                PhoneNumber.ToString() +
                ContactName.ToString() +
                DropShipFlag.ToString() +
                RequiredDeliveryDate.ToString() +
                ExteriorColor.ToString() +
                TenderTime.ToString() +
                Filler.ToString();
        }

        public static G07Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G07Detail rtn = new G07Detail();
            DateTime temp;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.AllocationNumber.Value = detailLine.Substring(rtn.AllocationNumber.Offset, rtn.AllocationNumber.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.TenderDate.Offset, rtn.TenderDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.TenderDate.Value = temp;

            rtn.OriginCode.Value = detailLine.Substring(rtn.OriginCode.Offset, rtn.OriginCode.Length).Trim();
            rtn.BayLocation.Value = detailLine.Substring(rtn.BayLocation.Offset, rtn.BayLocation.Length).Trim();
            rtn.TruckDestinationDealerCode.Value = detailLine.Substring(rtn.TruckDestinationDealerCode.Offset, 
                rtn.TruckDestinationDealerCode.Length).Trim();
            rtn.ShipToAddress1.Value = detailLine.Substring(rtn.ShipToAddress1.Offset, rtn.ShipToAddress1.Length).Trim();
            rtn.ShipToAddress2.Value = detailLine.Substring(rtn.ShipToAddress2.Offset, rtn.ShipToAddress2.Length).Trim();
            rtn.ShipToAddress3.Value = detailLine.Substring(rtn.ShipToAddress3.Offset, rtn.ShipToAddress3.Length).Trim();
            rtn.ZipCode.Value = detailLine.Substring(rtn.ZipCode.Offset, rtn.ZipCode.Length).Trim();
            rtn.PhoneNumber.Value = detailLine.Substring(rtn.PhoneNumber.Offset, rtn.PhoneNumber.Length).Trim();
            rtn.ContactName.Value = detailLine.Substring(rtn.ContactName.Offset, rtn.ContactName.Length).Trim();

            string boolTest = detailLine.Substring(rtn.DropShipFlag.Offset, rtn.DropShipFlag.Length).Trim();
            rtn.DropShipFlag.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;

            if (!DateTime.TryParseExact(detailLine.Substring(rtn.RequiredDeliveryDate.Offset, rtn.RequiredDeliveryDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp))
                temp = DateTime.Now.AddDays(3);
            rtn.RequiredDeliveryDate.Value = temp;

            rtn.ExteriorColor.Value = detailLine.Substring(rtn.ExteriorColor.Offset, rtn.ExteriorColor.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.TenderTime.Offset, rtn.TenderTime.Length).Trim(),
                "HHmmss", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.TenderTime.Value = temp;

            return rtn;
        }

        #region IComparable<G07Detail> Members

        public int CompareTo(G07Detail other)
        {
            int vinComp = this.VIN.Value.CompareTo(other.VIN.Value);
            int dropComp = this.TruckDestinationDealerCode.Value.CompareTo(other.TruckDestinationDealerCode.Value);

            if (vinComp == 0 && dropComp == 0)
                return 0;
            else if (vinComp < 0)
                return -1;
            else if (vinComp > 0)
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

    public class G07Detail_Comparer : IComparer<G07Detail>
    {
        #region IComparer<G07Detail> Members

        public int Compare(G07Detail x, G07Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G07Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G07Trailer()
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

        public static G07Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G07Trailer rtn = new G07Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
