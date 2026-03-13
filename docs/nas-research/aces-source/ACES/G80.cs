using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Dealer Records
    /// </summary>
    public class G80
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G80Detail> _detail = new List<G80Detail>();
        public G80Header Header { get; set; }
        public G80Trailer Trailer { get; set; }
        public List<G80Detail> Detail
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

        public G80()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G80Header();
            Trailer = new G80Trailer();
        }

        public static G80 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G80 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G80 rtn = new G80();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G80Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            //bool movedToPending = false;
            string detailErrorListing = string.Empty;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "DRT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G80Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            //string body = rtn.TransmissionInfo.LocalFile.Name + Environment.NewLine + Environment.NewLine;
                            //body += fvEx.Message + " (Header)";
                            //Utils.SendEmail("ACES Incoming File Validation Exception", body);
                            //if (!movedToPending && moveOnError)
                            //{
                            //    Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
                            //    movedToPending = true;
                            //}
                        }
                        continue;
                    case "DRT01":
                        G80Detail d = null;
                        try
                        {
                            d = G80Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            //detailErrorListing += line;
                            
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
                            rtn.Trailer = G80Trailer.Load(line);
                        }
                        catch (RecordCountMismatch rcmEx)
                        {
                            //string body = rtn.TransmissionInfo.LocalFile.Name + Environment.NewLine + Environment.NewLine;
                            //body += rcmEx.Message + " (Trailer)" + Environment.NewLine + Environment.NewLine;
                            //body += string.Format("Header count: {0}\tTrailer count: {1}\tActual count: {2}",
                            //    rtn.Header.TotalRecordCount.Value - 2,
                            //    rtn.Trailer.TransmitRecordCount.Value,
                            //    rtn.Detail.Count);
                            //Utils.SendEmail("ACES Record Count Mismatch Exception", body);
                            //if (!movedToPending && moveOnError)
                            //{
                            //    Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
                            //    movedToPending = true;
                            //}
                        }
                        break;
                }
                break;
            }

            //if (!detailErrorListing.Equals(string.Empty))
            //{
            //    //detailErrorListing = rtn.TransmissionInfo.LocalFile.Name + Environment.NewLine + Environment.NewLine + detailErrorListing;
            //    //Utils.SendEmail("ACES Incoming File Validation Exception", detailErrorListing);
            //    if (!movedToPending && moveOnError)
            //    {
            //        Utils.MovePendingFile(rtn.TransmissionInfo.LocalFile);
            //        movedToPending = true;
            //    }
            //}

            return rtn;
        }

        public bool Process(ATLDbDataContext db)
        {
            DateTime creation = CreatedDateTime;
            string type = "";
            switch (Header.CustomerCode.ToString())
            {
                case "HMA":
                    type = "Hyundai";
                    break;
                case "KMA":
                    type = "Kia";
                    break;
            }

            try
            {
                foreach (G80Detail det in Detail)
                {
                    db.sp_insert_g00(
                        det.DealerName.Value,
                        det.DealerCode.Value,
                        det.StreetAddress1.Value,
                        det.StreetAddress2.Value,
                        det.City.Value,
                        det.State.Value,
                        det.ZipCode.Value,
                        det.PhoneNumber.Value,
                        det.FaxNumber.Value,
                        det.DealerCode.Value,
                        type);
                }

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

    public class G80Header
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

        public G80Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "DRT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G80", Required = true };
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

        public static G80Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G80Header rtn = new G80Header();

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

    public class G80Detail : IComparable<G80Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> DealerCode { get; set; }
        public FixedPositionItem<string> DealerName { get; set; }
        public FixedPositionItem<string> DealerType { get; set; }
        public FixedPositionItem<string> ShipToContactName { get; set; }
        public FixedPositionItem<string> StreetAddress1 { get; set; }
        public FixedPositionItem<string> StreetAddress2 { get; set; }
        public FixedPositionItem<string> City { get; set; }
        public FixedPositionItem<string> State { get; set; }
        public FixedPositionItem<string> ZipCode { get; set; }
        public FixedPositionItem<string> PhoneNumber { get; set; }
        public FixedPositionItem<string> FaxNumber { get; set; }
        public FixedPositionItem<string> DistrictCode { get; set; }
        public FixedPositionItem<string> ADITVRatingCode { get; set; }
        public FixedPositionItem<bool> ActiveCode { get; set; } //Y/N
        public FixedPositionItem<string> RegionCode { get; set; }
        public FixedPositionItem<bool> DropShipFlag { get; set; }//Y/N
        public FixedPositionItem<string> Filler { get; private set; }
        public bool DoInsert { get; set; }

        public G80Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "DRT01", Required = true };
            DealerCode = new FixedPositionItem<string>() { Offset = 5, Length = 6, Value = string.Empty, Required = true };
            DealerName = new FixedPositionItem<string>() { Offset = 11, Length = 30, Value = string.Empty, Required = true };
            DealerType = new FixedPositionItem<string>() { Offset = 41, Length = 2, Value = string.Empty, Required = true };
            ShipToContactName = new FixedPositionItem<string>() { Offset = 43, Length = 30, Value = string.Empty, Required = true };
            StreetAddress1 = new FixedPositionItem<string>() { Offset = 73, Length = 30, Value = string.Empty, Required = true };
            StreetAddress2 = new FixedPositionItem<string>() { Offset = 103, Length = 30, Value = string.Empty };
            City = new FixedPositionItem<string>() { Offset = 133, Length = 30, Value = string.Empty, Required = true };
            State = new FixedPositionItem<string>() { Offset = 163, Length = 2, Value = string.Empty, Required = true };
            ZipCode = new FixedPositionItem<string>() { Offset = 165, Length = 10, Value = string.Empty, Required = true };
            PhoneNumber = new FixedPositionItem<string>() { Offset = 175, Length = 10, Value = string.Empty, Required = true };
            FaxNumber = new FixedPositionItem<string>() { Offset = 185, Length = 10, Value = string.Empty };
            DistrictCode = new FixedPositionItem<string>() { Offset = 195, Length = 5, Value = string.Empty };
            ADITVRatingCode = new FixedPositionItem<string>() { Offset = 200, Length = 4, Value = string.Empty };
            ActiveCode = new FixedPositionItem<bool>() { Offset = 204, Length = 1, Value = true, Format = "{0:Y;;N}", Required = true };
            RegionCode = new FixedPositionItem<string>() { Offset = 205, Length = 2, Value = string.Empty, Required = true };
            DropShipFlag = new FixedPositionItem<bool>() { Offset = 207, Length = 1, Value = false, Format = "{0:Y;;N}", Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 208, Length = 42, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                DealerCode.ToString() +
                DealerName.ToString() +
                DealerType.ToString() +
                ShipToContactName.ToString() +
                StreetAddress1.ToString() +
                StreetAddress2.ToString() +
                City.ToString() +
                State.ToString() +
                ZipCode.ToString() +
                PhoneNumber.ToString() +
                FaxNumber.ToString() +
                DistrictCode.ToString() +
                ADITVRatingCode.ToString() +
                ActiveCode.ToString() +
                RegionCode.ToString() +
                DropShipFlag.ToString() +
                Filler.ToString();
        }

        public static G80Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G80Detail rtn = new G80Detail();

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.DealerCode.Value = detailLine.Substring(rtn.DealerCode.Offset, rtn.DealerCode.Length).Trim();
            rtn.DealerName.Value = detailLine.Substring(rtn.DealerName.Offset, rtn.DealerName.Length).Trim();
            rtn.DealerType.Value = detailLine.Substring(rtn.DealerType.Offset, rtn.DealerType.Length).Trim();
            rtn.ShipToContactName.Value = detailLine.Substring(rtn.ShipToContactName.Offset, rtn.ShipToContactName.Length).Trim();
            rtn.StreetAddress1.Value = detailLine.Substring(rtn.StreetAddress1.Offset, rtn.StreetAddress1.Length).Trim();
            rtn.StreetAddress2.Value = detailLine.Substring(rtn.StreetAddress2.Offset, rtn.StreetAddress2.Length).Trim();
            rtn.City.Value = detailLine.Substring(rtn.City.Offset, rtn.City.Length).Trim();
            rtn.State.Value = detailLine.Substring(rtn.State.Offset, rtn.State.Length).Trim();
            rtn.ZipCode.Value = detailLine.Substring(rtn.ZipCode.Offset, rtn.ZipCode.Length).Trim();
            rtn.PhoneNumber.Value = detailLine.Substring(rtn.PhoneNumber.Offset, rtn.PhoneNumber.Length).Trim();
            rtn.FaxNumber.Value = detailLine.Substring(rtn.FaxNumber.Offset, rtn.FaxNumber.Length).Trim();
            rtn.DistrictCode.Value = detailLine.Substring(rtn.DistrictCode.Offset, rtn.DistrictCode.Length).Trim();
            rtn.ADITVRatingCode.Value = detailLine.Substring(rtn.ADITVRatingCode.Offset, rtn.ADITVRatingCode.Length).Trim();

            string boolTest = detailLine.Substring(rtn.ActiveCode.Offset, rtn.ActiveCode.Length).Trim();
            rtn.DropShipFlag.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;
            rtn.ActiveCode.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;
            rtn.RegionCode.Value = detailLine.Substring(rtn.RegionCode.Offset, rtn.RegionCode.Length).Trim();

            boolTest = detailLine.Substring(rtn.DropShipFlag.Offset, rtn.DropShipFlag.Length).Trim();
            rtn.DropShipFlag.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;

            return rtn;
        }

        #region IComparable<G80Detail> Members

        public int CompareTo(G80Detail other)
        {
            int codeComp = this.DealerCode.Value.CompareTo(other.DealerCode.Value);
            int nameComp = this.DealerName.Value.CompareTo(other.DealerName.Value);

            if (codeComp == 0 && nameComp == 0)
                return 0;
            else if (codeComp < 0)
                return -1;
            else if (codeComp > 0)
                return 1;
            else
            {
                if (nameComp < 0)
                    return -1;
                else if (nameComp > 0)
                    return 1;
                else return 0;
            }
        }

        #endregion
    }

    public class G80Detail_Comparer : IComparer<G80Detail>
    {
        #region IComparer<G80Detail> Members

        public int Compare(G80Detail x, G80Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G80Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G80Trailer()
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

        public static G80Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G80Trailer rtn = new G80Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
