using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.ComponentModel;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES VIN Shipment Status
    /// </summary>
    public class R41
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        public R41Header Header { get; set; }
        public R41Trailer Trailer { get; set; }
        public List<R41Detail> Detail { get; set; }

        public R41()
        {
            CreatedDateTime = DateTime.Now;
            Header = new R41Header(this);
            Trailer = new R41Trailer(this);
            Detail = new List<R41Detail>();
        }

        public static List<R41> Generate(ATLDbDataContext db, BackgroundWorker worker, DateTime time)
        {
            List<R41> rtn = new List<R41>();

            var q = from p in db.vw_edi_aces_r41s
                    select p;

            if (q.Count() < 1)
                return null;

            int total = q.Count();

            List<R41Detail> allDetail = new List<R41Detail>(total);

            ProcessProgress pp = new ProcessProgress();
            pp.TotalFiles = total;
            pp.FilesProcessed = 0;
            pp.State = OperationState.Query;

            foreach (vw_edi_aces_r41 r in q)
            {
                X85 x85 = (from p in db.X85s
                           where p.X85Id == r.X85Id
                           select p).FirstOrDefault() as X85;

                pp.FilesProcessed++;
                pp.StatusMessage = string.Format("Generating R41 ({0} of {1})...", pp.FilesProcessed, total);
                worker.ReportProgress(Utils.CalculatePercentage(pp.FilesProcessed, total), pp);

                R41Detail d = new R41Detail();

                if (r.SenderId == null || r.SenderId.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "SenderId blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "SenderId is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: SenderId blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.SenderId = r.SenderId;
                if (r.MFGCode == null || r.MFGCode.Trim().Equals(""))
                {
                    Error err = new Error();
                    err.Message = "MfgCode blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "MfgCode is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: MfgCode blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                switch (r.MFGCode.Trim().ToUpper())
                {
                    case "HYUNDAI":
                        d.CustomerCode = "HMA";
                        break;
                    case "KIA":
                        d.CustomerCode = "KMA";
                        break;
                }
                //d.CustomerCode = (r.MFGCode.Trim().Substring(0, 1).ToUpper().Equals("H") ? "HMA" : "KMA");
                d.PortCode = (r.VPCCode != null ? r.VPCCode.Trim() : "");
                d.BillOfLadingNum.Value = r.D20Id.ToString();
                if (d.BillOfLadingNum.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Bill of Lading blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Bill of Lading Number is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: Bill of Lading Number blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.DamageIndicator.Value = false;
                d.DeliveryStatusCode.Value = r.Delivery_Status_Code.Trim();
                if (d.DeliveryStatusCode.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Delivery Status Code blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Delivery Status Code is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: Delivery Status Code blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.DestinationCode.Value = r.DropCustNumber.Trim();
                if (d.DestinationCode.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Destination Code blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Destination Code is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: Destination Code blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.LocationCode.Value = r.SPLC_Zip_Code.Trim();
                d.OriginCode.Value = r.OriginCode.Trim();
                d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim();
                if (d.ShipmentAuthorizationCode.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Shipment Authorization Code blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Shipment Authorization Code is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: Shipment Authorization Code blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.SPLCTransmissionFlag.Value =
                    (r.SPLC_Transmission_Flag == 'T' ? true : false);
                if (r.StatusDateTime == null)
                {
                    Error err = new Error();
                    err.Message = "Status DateTime blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Status DateTime is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: Status Date/Time blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.StatusDate.Value = Convert.ToDateTime(r.StatusDateTime);
                d.StatusTime.Value = Convert.ToDateTime(r.StatusDateTime);
                d.VIN.Value = r.VIN.Trim();
                if (d.VIN.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "VIN blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R41";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "VIN is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R41: VIN blank for X85Id {0}", r.X85Id);
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.X85Id = r.X85Id;
                d.X85 = x85;
                d.X00Id = r.X00Id.Value;

                x85.ProcessedTimeString = time;
                x85.Processed = 2;
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);

                allDetail.Add(d);
            }

            allDetail.Sort(new R41Detail_Comparer());
            R41Detail prev = null;
            bool headerSet = false;
            R41 temp = new R41();
            for (int i = 0; i < allDetail.Count; i++)
            {
                R41Detail d = allDetail[i];

                if (prev != null && prev.CompareTo(d) != 0)
                {
                    rtn.Add(temp);
                    temp = new R41();
                    headerSet = false;
                }

                if (!headerSet)
                {
                    temp.Header.CustomerCode.Value = d.CustomerCode;
                    temp.Header.PortCode.Value = d.PortCode;
                    temp.CreatedDateTime = time;
                    temp.Header.TransmissionDate.Value = time;
                    temp.Header.TransmissionTime.Value = time;
                    temp.Header.SenderID.Value = d.SenderId;
                    temp.X00Id = d.X00Id;
                    headerSet = true;
                }

                d.Parent = temp;
                temp.Detail.Add(d);
                prev = d;

                if (i == allDetail.Count - 1)
                {
                    rtn.Add(temp);
                }
            }

            #region old
            //bool headerSet = false;

            //R41 rtn = new R41();
            //rtn.CreatedDateTime = time;
            //rtn.Header.TransmissionDate.Value = time;
            //rtn.Header.TransmissionTime.Value = time;

            //var q = from p in db.vw_edi_aces_r41s
            //        select p;

            //if (q.Count() < 1)
            //    return null;

            //int total = q.Count();

            //ProcessProgress pp = new ProcessProgress();
            //pp.TotalFiles = total;
            //pp.FilesProcessed = 0;
            //pp.State = OperationState.Query;

            //foreach (vw_edi_aces_r41 r in q)
            //{
            //    X85 x85 = (from p in db.X85s
            //               where p.X85Id == r.X85Id
            //               select p).FirstOrDefault() as X85;

            //    pp.FilesProcessed++;
            //    pp.StatusMessage = string.Format("Generating R41 ({0} of {1})...", pp.FilesProcessed, total);
            //    worker.ReportProgress(Utils.CalculatePercentage(pp.FilesProcessed, total), pp);

            //    if (!headerSet)
            //    {
            //        rtn.Header.SenderID.Value = r.SenderId;
            //        if (r.MFGCode.Trim().Substring(0, 1).ToLowerInvariant().Equals("K"))
            //        {
            //            rtn.Header.CustomerCode.Value = "KMA";
            //        }
            //        else
            //        {
            //            rtn.Header.CustomerCode.Value = "HMA";
            //        }
            //        rtn.Header.PortCode.Value = r.VPCCode.Trim();
            //        headerSet = true;
            //    }

            //    R41Detail d = new R41Detail(rtn);
            //    d.BillOfLadingNum.Value = r.D20Id.ToString();
            //    if (d.BillOfLadingNum.Value.Equals(""))
            //    {
            //        string err = string.Format("R41: Bill of Lading Number blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.DamageIndicator.Value = false;
            //    d.DeliveryStatusCode.Value = r.Delivery_Status_Code.Trim();
            //    if (d.DeliveryStatusCode.Value.Equals(""))
            //    {
            //        string err = string.Format("R41: Delivery Status Code blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.DestinationCode.Value = r.DropCustNumber.Trim();
            //    if (d.DestinationCode.Value.Equals(""))
            //    {
            //        string err = string.Format("R41: Destination Code blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.LocationCode.Value = r.SPLC_Zip_Code.Trim();
            //    d.OriginCode.Value = r.OriginCode.Trim();
            //    d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim();
            //    if (d.ShipmentAuthorizationCode.Value.Equals(""))
            //    {
            //        string err = string.Format("R41: Shipment Authorization Code blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.SPLCTransmissionFlag.Value =
            //        (r.SPLC_Transmission_Flag == 'T' ? true : false);
            //    if (r.StatusDateTime == null)
            //    {
            //        string err = string.Format("R41: Status Date/Time blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.StatusDate.Value = Convert.ToDateTime(r.StatusDateTime);
            //    d.StatusTime.Value = Convert.ToDateTime(r.StatusDateTime);
            //    d.VIN.Value = r.VIN.Trim();
            //    if (d.VIN.Value.Equals(""))
            //    {
            //        string err = string.Format("R41: VIN blank for X85Id {0}", r.X85Id);
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.X85Id = r.X85Id;
            //    d.X85 = x85;
            //    rtn.X00Id = (int)r.X00Id;

            //    rtn.Detail.Add(d);

            //    x85.ProcessedTimeString = time;
            //    x85.Processed = 2;
            //    db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
            //}
            #endregion

            return rtn;
        }

        public void Save(string path)
        {
            FileInfo info = new FileInfo(path);
            int count = 2;
            while (info.Exists)
            {
                path = path.Split(',')[0] + "(" + count.ToString() + ")" + info.Extension;
                count++;
                info = new FileInfo(path);
            }

            FileStream fs = null;
            StreamWriter sw = null;
            try
            {
                if (!info.Directory.Exists)
                    info.Directory.Create();
                fs = info.Create();
                sw = new StreamWriter(fs);

                sw.WriteLine(Header);
                foreach (R41Detail d in Detail)
                    sw.WriteLine(d);
                sw.Write(Trailer);
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }
            finally
            {
                sw.Close();
            }

        }

        public string GetFileName(int dicn)
        {
            return string.Format("{0}{1:0000000}.txt",
                "R41",
                dicn);
        }
    }

    public class R41Header
    {
        public R41 Parent { get; set; }
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> SenderID { get; set; }
        public FixedPositionItem<string> ReceiverID { get; set; }
        public FixedPositionItem<string> TransmissionID { get; set; }
        public FixedPositionItem<DateTime> TransmissionDate { get; set; }
        public FixedPositionItem<DateTime> TransmissionTime { get; set; }
        public FixedPositionItem<string> PortCode { get; set; }
        public FixedPositionItem<string> CustomerCode { get; set; }
        private FixedPositionItem<int> _totalRecordCount { get; set; }
        public FixedPositionItem<int> TotalRecordCount
        {
            get
            {
                if (Parent != null && Parent.Detail != null)
                    _totalRecordCount.Value = Parent.Detail.Count + 2;
                return _totalRecordCount;
            }
            private set
            {
                _totalRecordCount = value;
            }
        }
        public FixedPositionItem<string> Filler { get; private set; }

        public R41Header()
        {
            Init();
        }

        public R41Header(R41 parent)
        {
            Parent = parent;
            Init();
        }

        private void Init()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "SST00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = string.Empty, Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = "ACE", Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "R41", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Value = DateTime.Now, Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Value = DateTime.Now, Required = true };
            PortCode = new FixedPositionItem<string>() { Offset = 28, Length = 2, Value = string.Empty };
            CustomerCode = new FixedPositionItem<string>() { Offset = 30, Length = 10, Value = string.Empty };
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
    }

    public class R41Detail : IComparable<R41Detail>
    {
        public R41 Parent { get; set; }
        public int X85Id { get; set; }
        public X85 X85 { get; set; }
        public int X00Id { get; set; }
        public string CustomerCode { get; set; }
        public string PortCode { get; set; }
        public string SenderId { get; set; }
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> BillOfLadingNum { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<DateTime> StatusDate { get; set; }
        public FixedPositionItem<DateTime> StatusTime { get; set; }
        public FixedPositionItem<string> DeliveryStatusCode { get; set; }
        public FixedPositionItem<string> LocationCode { get; set; }
        public FixedPositionItem<string> Filler1 { get; set; }
        public FixedPositionItem<string> OriginCode { get; set; }
        public FixedPositionItem<string> DestinationCode { get; set; }
        public FixedPositionItem<string> TruckType { get; set; }
        public FixedPositionItem<bool> DamageIndicator { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<bool> SPLCTransmissionFlag { get; set; }
        public FixedPositionItem<string> Filler2 { get; set; }

        public R41Detail()
        {
            Init();
        }

        public R41Detail(R41 parent)
        {
            Parent = parent;
            Init();
        }

        private void Init()
        {
            CustomerCode = string.Empty;
            PortCode = string.Empty;
            SenderId = string.Empty;
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "SST01", Required = true };
            BillOfLadingNum = new FixedPositionItem<string>() { Offset = 5, Length = 15, Value = string.Empty, Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 20, Length = 17, Value = string.Empty, Required = true };
            StatusDate = new FixedPositionItem<DateTime>() { Offset = 37, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            StatusTime = new FixedPositionItem<DateTime>() { Offset = 45, Length = 6, Format = "{0:HHmmss}", Required = true };
            DeliveryStatusCode = new FixedPositionItem<string>() { Offset = 51, Length = 3, Value = string.Empty, Required = true };
            LocationCode = new FixedPositionItem<string>() { Offset = 54, Length = 10, Value = string.Empty };
            Filler1 = new FixedPositionItem<string>() { Offset = 64, Length = 1, Value = " ", Required = true };
            OriginCode = new FixedPositionItem<string>() { Offset = 65, Length = 7, Value = string.Empty, Required = true };
            DestinationCode = new FixedPositionItem<string>() { Offset = 72, Length = 7, Value = string.Empty, Required = true };
            TruckType = new FixedPositionItem<string>() { Offset = 79, Length = 1, Value = string.Empty };
            DamageIndicator = new FixedPositionItem<bool>() { Offset = 80, Length = 1, Value = false, Format = "{0:Y;;N}", Required = false };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 81, Length = 12, Value = string.Empty, Required = true };
            SPLCTransmissionFlag = new FixedPositionItem<bool>() { Offset = 93, Length = 1, Value = false, Format = "{0:T;;F}", Required = false };
            Filler2 = new FixedPositionItem<string>() { Offset = 94, Length = 156, Value = new string(Utils.FillerChar, 156), Required = false };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                BillOfLadingNum.ToString() +
                VIN.ToString() +
                StatusDate.ToString() +
                StatusTime.ToString() +
                DeliveryStatusCode.ToString() +
                LocationCode.ToString() +
                Filler1.ToString() +
                OriginCode.ToString() +
                DestinationCode.ToString() +
                TruckType.ToString() +
                DamageIndicator.ToString() +
                ShipmentAuthorizationCode.ToString() +
                SPLCTransmissionFlag.ToString() +
                Filler2.ToString();
        }

        #region IComparable<R41Detail> Members

        public int CompareTo(R41Detail other)
        {
            int cComp = this.CustomerCode.CompareTo(other.CustomerCode);
            int pComp = this.PortCode.CompareTo(other.PortCode);

            if (cComp != 0)
                return cComp;
            else if (cComp == 0 && pComp != 0)
                return pComp;
            else
                return 0;
        }

        #endregion
    }

    public class R41Detail_Comparer : IComparer<R41Detail>
    {
        #region IComparer<R41Detail> Members

        public int Compare(R41Detail x, R41Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class R41Trailer
    {
        public R41 Parent { get; set; }
        public FixedPositionItem<string> RecordID { get; set; }
        private FixedPositionItem<int> _transmitRecordCount { get; set; }
        public FixedPositionItem<int> TransmitRecordCount
        {
            get
            {
                if (Parent != null && Parent.Detail != null)
                    _transmitRecordCount.Value = Parent.Detail.Count;
                return _transmitRecordCount;
            }
            private set
            {
                _transmitRecordCount = value;
            }
        }
        public FixedPositionItem<string> Filler { get; set; }

        public R41Trailer()
        {
            Init();
        }

        public R41Trailer(R41 parent)
        {
            Parent = parent;
            Init();
        }

        private void Init()
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

    }
}
