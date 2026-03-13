using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Rail Shipment
    /// </summary>
    public class G08
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G08Detail> _detail = new List<G08Detail>();
        public G08Header Header { get; set; }
        public G08Trailer Trailer { get; set; }
        public List<G08Detail> Detail
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

        public G08()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G08Header();
            Trailer = new G08Trailer();
        }

        public static G08 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G08 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G08 rtn = new G08();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G08Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "RST00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G08Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G08";
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
                    case "RST01":
                        G08Detail d = null;
                        try
                        {
                            d = G08Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Detail)";
                            err.Description = "Missing required information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G08";
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
                            rtn.Trailer = G08Trailer.Load(line);
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
                            err.EdiSet = "G08";
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

            //Detail.Sort(new G08Detail_Comparer());
            return false;
        }
    }

    public class G08Header
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

        public G08Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "RST00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G08", Required = true };
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

        public static G08Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G08Header rtn = new G08Header();

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

    public class G08Detail : IComparable<G08Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> RRWaybillNumber { get; set; }
        public FixedPositionItem<string> RailVendorExPort { get; set; }
        public FixedPositionItem<string> RailCarID { get; set; }
        public FixedPositionItem<string> Filler1 { get; set; }
        public FixedPositionItem<string> ExteriorColor { get; set; }
        public FixedPositionItem<string> RampDestinationCode { get; set; }
        public FixedPositionItem<string> DestinationDealerCode { get; set; }
        public FixedPositionItem<bool> DropShipFlag { get; set; }
        public FixedPositionItem<string> ShipToAddress1 { get; set; }
        public FixedPositionItem<string> ShipToAddress2 { get; set; }
        public FixedPositionItem<string> ShipToAddress3 { get; set; }
        public FixedPositionItem<string> ZipCode { get; set; }
        public FixedPositionItem<string> PhoneNumber { get; set; }
        public FixedPositionItem<string> ContactName { get; set; }
        public FixedPositionItem<DateTime> RampArrivalETA { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<DateTime> RailShipDate { get; set; }
        public FixedPositionItem<string> RampOrigin { get; set; }
        public FixedPositionItem<string> FinalTruckVendorFromRamp { get; set; }
        public FixedPositionItem<string> Filler2 { get; set; }
        public bool DoInsert { get; set; }

        public G08Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "RST01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            RRWaybillNumber = new FixedPositionItem<string>() { Offset = 22, Length = 6, Value = string.Empty, Required = true };
            RailVendorExPort = new FixedPositionItem<string>() { Offset = 28, Length = 5, Value = string.Empty, Required = true };
            RailCarID = new FixedPositionItem<string>() { Offset = 33, Length = 10, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 43, Length = 1, Value = string.Empty };
            ExteriorColor = new FixedPositionItem<string>() { Offset = 44, Length = 3, Value = string.Empty, Required = true };
            RampDestinationCode = new FixedPositionItem<string>() { Offset = 47, Length = 5, Value = string.Empty, Required = true };
            DestinationDealerCode = new FixedPositionItem<string>() { Offset = 52, Length = 7, Value = string.Empty, Required = true };
            DropShipFlag = new FixedPositionItem<bool>() { Offset = 59, Length = 1, Value = false, Format = "{0:Y;;N}", Required = true };
            ShipToAddress1 = new FixedPositionItem<string>() { Offset = 60, Length = 30, Value = string.Empty };
            ShipToAddress2 = new FixedPositionItem<string>() { Offset = 90, Length = 30, Value = string.Empty };
            ShipToAddress3 = new FixedPositionItem<string>() { Offset = 120, Length = 30, Value = string.Empty };
            ZipCode = new FixedPositionItem<string>() { Offset = 150, Length = 10, Value = string.Empty };
            PhoneNumber = new FixedPositionItem<string>() { Offset = 160, Length = 20, Value = string.Empty };
            ContactName = new FixedPositionItem<string>() { Offset = 180, Length = 30, Value = string.Empty };
            RampArrivalETA = new FixedPositionItem<DateTime>() { Offset = 210, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 218, Length = 12, Value = string.Empty, Required = true };
            RailShipDate = new FixedPositionItem<DateTime>() { Offset = 230, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}" };
            RampOrigin = new FixedPositionItem<string>() { Offset = 238, Length = 5, Value = string.Empty, Required = true };
            FinalTruckVendorFromRamp = new FixedPositionItem<string>() { Offset = 243, Length = 5, Value = string.Empty, Required = true };
            Filler2 = new FixedPositionItem<string>() { Offset = 248, Length = 2, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                RRWaybillNumber.ToString() +
                RailVendorExPort.ToString() +
                RailCarID.ToString() +
                Filler1.ToString() +
                ExteriorColor.ToString() +
                RampDestinationCode.ToString() +
                DestinationDealerCode.ToString() +
                DropShipFlag.ToString() +
                ShipToAddress1.ToString() +
                ShipToAddress2.ToString() +
                ShipToAddress3.ToString() +
                ZipCode.ToString() +
                PhoneNumber.ToString() +
                ContactName.ToString() +
                RampArrivalETA.ToString() +
                ShipmentAuthorizationCode.ToString() +
                RailShipDate.ToString() +
                RampOrigin.ToString() +
                FinalTruckVendorFromRamp.ToString() +
                Filler2.ToString();
        }

        public static G08Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G08Detail rtn = new G08Detail();
            DateTime temp;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.RRWaybillNumber.Value = detailLine.Substring(rtn.RRWaybillNumber.Offset, rtn.RRWaybillNumber.Length).Trim();
            rtn.RailVendorExPort.Value = detailLine.Substring(rtn.RailVendorExPort.Offset, rtn.RailVendorExPort.Length).Trim();
            rtn.RailCarID.Value = detailLine.Substring(rtn.RailCarID.Offset, rtn.RailCarID.Length).Trim();
            rtn.RampDestinationCode.Value = detailLine.Substring(rtn.RampDestinationCode.Offset, rtn.RampDestinationCode.Length).Trim();
            rtn.DestinationDealerCode.Value = detailLine.Substring(rtn.DestinationDealerCode.Offset, rtn.DestinationDealerCode.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.RampArrivalETA.Offset, rtn.RampArrivalETA.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.RampArrivalETA.Value = temp;

            rtn.ShipmentAuthorizationCode.Value = detailLine.Substring(rtn.ShipmentAuthorizationCode.Offset, 
                rtn.ShipmentAuthorizationCode.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.RailShipDate.Offset, rtn.RailShipDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.RailShipDate.Value = temp;

            rtn.RampOrigin.Value = detailLine.Substring(rtn.RampOrigin.Offset, rtn.RampOrigin.Length).Trim();
            rtn.FinalTruckVendorFromRamp.Value = detailLine.Substring(rtn.FinalTruckVendorFromRamp.Offset,
                rtn.FinalTruckVendorFromRamp.Length).Trim();
            rtn.ShipToAddress1.Value = detailLine.Substring(rtn.ShipToAddress1.Offset, rtn.ShipToAddress1.Length).Trim();
            rtn.ShipToAddress2.Value = detailLine.Substring(rtn.ShipToAddress2.Offset, rtn.ShipToAddress2.Length).Trim();
            rtn.ShipToAddress3.Value = detailLine.Substring(rtn.ShipToAddress3.Offset, rtn.ShipToAddress3.Length).Trim();
            rtn.ZipCode.Value = detailLine.Substring(rtn.ZipCode.Offset, rtn.ZipCode.Length).Trim();
            rtn.PhoneNumber.Value = detailLine.Substring(rtn.PhoneNumber.Offset, rtn.PhoneNumber.Length).Trim();
            rtn.ContactName.Value = detailLine.Substring(rtn.ContactName.Offset, rtn.ContactName.Length).Trim();

            string boolTest = detailLine.Substring(rtn.DropShipFlag.Offset, rtn.DropShipFlag.Length).Trim();
            rtn.DropShipFlag.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;

            rtn.ExteriorColor.Value = detailLine.Substring(rtn.ExteriorColor.Offset, rtn.ExteriorColor.Length).Trim();

            return rtn;
        }

        #region IComparable<G08Detail> Members

        public int CompareTo(G08Detail other)
        {
            int vinComp = this.VIN.Value.CompareTo(other.VIN.Value);
            int dropComp = this.DestinationDealerCode.Value.CompareTo(other.DestinationDealerCode.Value);

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

    public class G08Detail_Comparer : IComparer<G08Detail>
    {
        #region IComparer<G08Detail> Members

        public int Compare(G08Detail x, G08Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G08Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G08Trailer()
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

        public static G08Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G08Trailer rtn = new G08Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
