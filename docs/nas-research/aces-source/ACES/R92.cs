using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.ComponentModel;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Invoice
    /// </summary>
    public class R92
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        public R92Header Header { get; set; }
        public R92Trailer Trailer { get; set; }
        public List<R92Detail> Detail { get; set; }

        public R92()
        {
            CreatedDateTime = DateTime.Now;
            Header = new R92Header(this);
            Trailer = new R92Trailer(this);
            Detail = new List<R92Detail>();
        }

        public static List<R92> Generate(ATLDbDataContext db, BackgroundWorker worker, DateTime time)
        {
            List<R92> rtn = new List<R92>();

            var q = from p in db.vw_edi_aces_r92s
                    select p;

            if (q.Count() < 1)
                return null;

            int total = q.Count();

            List<R92Detail> allDetail = new List<R92Detail>(total);

            ProcessProgress pp = new ProcessProgress();
            pp.TotalFiles = total;
            pp.FilesProcessed = 0;
            pp.State = OperationState.Query;

            foreach (vw_edi_aces_r92 r in q)
            {
                X85 x85 = (from p in db.X85s
                           where p.X85Id == r.X85Id
                           select p).FirstOrDefault() as X85;

                if (x85.Processed == 1)
                    continue;

                pp.FilesProcessed++;
                pp.StatusMessage = string.Format("Generating R92 ({0} of {1})...", pp.FilesProcessed, total);
                worker.ReportProgress(Utils.CalculatePercentage(pp.FilesProcessed, total), pp);

                R92Detail d = new R92Detail();
                if (r.SenderId == null || r.SenderId.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "SenderId blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "SenderId is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: SenderId blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.SenderId = r.SenderId;
                if (r.MfgCode == null || r.MfgCode.Trim().Equals(""))
                {
                    Error err = new Error();
                    err.Message = "MfgCode blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "MfgCode is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: MfgCode blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                switch (r.MfgCode.Trim().ToUpper())
                {
                    case "HYUNDAI":
                        d.CustomerCode = "HMA";
                        break;
                    case "KIA":
                        d.CustomerCode = "KMA";
                        break;
                }
                //d.CustomerCode = (r.MfgCode.Trim().Substring(0, 1).ToUpper().Equals("H") ? "HMA" : "KMA");
                d.PortCode = (r.VPCCode != null ? r.VPCCode.Trim() : "");
                if (r.DropTimeString == null)
                {
                    Error err = new Error();
                    err.Message = "Drop DateTime blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Drop DateTime is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: Drop Date/Time blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.CompletionDate.Value = Convert.ToDateTime(r.DropTimeString);
                d.CompletionTime.Value = d.CompletionDate.Value;
                if (r.TransportAmount == null)
                {
                    Error err = new Error();
                    err.Message = "Credit Amount blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Credit Amount is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: Credit Amount NULL for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.CreditAmount.Value = Convert.ToDecimal(r.TransportAmount) * 100;
                d.CreditSign.Value = (d.CreditAmount.Value >= 0) ? '+' : '-';
                d.GLAccountNumber.Value = r.CarrierCode.Trim(); //"1405";
                if (r.InvoiceDate == null)
                {
                    Error err = new Error();
                    err.Message = "Invoice DateTime blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Invoice DateTime is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: Invoice Date NULL for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.InvoiceDate.Value = Convert.ToDateTime(r.InvoiceDate);
                d.InvoiceNumber.Value = r.InvoiceNumber.Trim();
                if (d.InvoiceNumber.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Invoice Number blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Invoice Number is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: Invoice Number blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.ManualInvoiceFlag.Value = false;
                d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim(); //ADX107772
                if (d.ShipmentAuthorizationCode.Value.Equals(""))
                {
                    Error err = new Error();
                    err.Message = "Shipment Authorization Code blank";
                    err.VIN = r.VIN.Trim();
                    err.EdiSet = "R92";
                    err.X85Id = r.X85Id;
                    err.ErrorDateTime = time;
                    err.Active = true;
                    err.Description = "Shipment Authorization Code is required to send to ACES";
                    err.System = "ACES";
                    err.Code = "ACES_VALIDATION_EXCEPTION";
                    //string err = string.Format("R92: Shipment Authorization Code blank for VIN # {0}", r.VIN.Trim());
                    Utils.AddErrorEntry(err);
                    continue;
                }
                d.DestinationCode.Value = r.DropCustNumber.Trim();
                d.OriginCode.Value = r.OriginCode.Trim();
                d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim();
                d.VIN.Value = r.VIN.Trim();
                d.X85Id = r.X85Id;
                d.X85 = x85;
                d.X00Id = r.X00Id.Value;

                x85.ProcessedTimeString = time;
                x85.Processed = 2;
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);

                allDetail.Add(d);
            }

            allDetail.Sort(new R92Detail_Comparer());
            R92Detail prev = null;
            bool headerSet = false;
            R92 temp = new R92();
            for(int i=0; i<allDetail.Count; i++)
            {
                R92Detail d = allDetail[i];

                if (prev != null && prev.CompareTo(d) != 0)
                {
                    rtn.Add(temp);
                    temp = new R92();
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
            //R92 temp = null;
            //List<R92> rtn = new List<R92>();

            //var q = from p in db.vw_edi_aces_r92s
            //        select p;

            //if (q.Count() < 1)
            //    return null;

            //int total = q.Count();

            //ProcessProgress pp = new ProcessProgress();
            //pp.TotalFiles = total;
            //pp.FilesProcessed = 0;
            //pp.State = OperationState.Query;

            //foreach (vw_edi_aces_r92 r in q)
            //{
            //    X85 x85 = (from p in db.X85s
            //               where p.X85Id == r.X85Id
            //               select p).FirstOrDefault() as X85;

            //    if (x85.Processed == 1)
            //        continue;

            //    pp.FilesProcessed++;
            //    pp.StatusMessage = string.Format("Generating R92 ({0} of {1})...", pp.FilesProcessed, total);
            //    worker.ReportProgress(Utils.CalculatePercentage(pp.FilesProcessed, total), pp);

            //    if (!headerSet)
            //    {
            //        rtn.CreatedDateTime = time;
            //        rtn.Header.TransmissionDate.Value = time;
            //        rtn.Header.TransmissionTime.Value = time;
            //        rtn.Header.SenderID.Value = r.SenderId;

            //        if (r.MfgCode.Trim().Substring(0, 1).ToUpperInvariant().Equals("K"))
            //        {
            //            rtn.Header.CustomerCode.Value = "KMA";
            //        }
            //        else if(r.MfgCode.Trim().Substring(0,1).ToUpperInvariant().Equals("H"))
            //        {
            //            rtn.Header.CustomerCode.Value = "HMA";
            //        }
            //        rtn.Header.PortCode.Value = r.VPCCode.Trim();
            //        headerSet = true;
            //    }

            //    R92Detail d = new R92Detail(rtn);
            //    d.Parent = rtn;
            //    if (r.DropTimeString == null)
            //    {
            //        string err = string.Format("R92: Drop Date/Time blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.CompletionDate.Value = Convert.ToDateTime(r.DropTimeString);
            //    d.CompletionTime.Value = d.CompletionDate.Value;
            //    if (r.TransportAmount == null)
            //    {
            //        string err = string.Format("R92: Credit Amount NULL for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.CreditAmount.Value = Convert.ToDecimal(r.TransportAmount) * 100;
            //    d.CreditSign.Value = (d.CreditAmount.Value >= 0) ? '+' : '-';
            //    d.GLAccountNumber.Value = r.CarrierCode.Trim(); //"1405";
            //    if (r.InvoiceDate == null)
            //    {
            //        string err = string.Format("R92: Invoice Date NULL for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.InvoiceDate.Value = Convert.ToDateTime(r.InvoiceDate);
            //    d.InvoiceNumber.Value = r.InvoiceNumber.Trim();
            //    if (d.InvoiceNumber.Value.Equals(""))
            //    {
            //        string err = string.Format("R92: Invoice Number blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.ManualInvoiceFlag.Value = false;
            //    d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim(); //ADX107772
            //    if (d.ShipmentAuthorizationCode.Value.Equals(""))
            //    {
            //        string err = string.Format("R92: Shipment Authorization Code blank for VIN # {0}", r.VIN.Trim());
            //        Utils.AddErrorEntry(err);
            //        continue;
            //    }
            //    d.DestinationCode.Value = r.DropCustNumber.Trim();
            //    d.OriginCode.Value = r.OriginCode.Trim();
            //    d.ShipmentAuthorizationCode.Value = r.AuthorizationCode.Trim();
            //    d.VIN.Value = r.VIN.Trim();
            //    d.X85Id = r.X85Id;
            //    d.X85 = x85;
            //    rtn.X00Id = Convert.ToInt32(r.X00Id);

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
                foreach (R92Detail d in Detail)
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
                "R92",
                dicn);
        }
    }

    public class R92Header
    {
        public R92 Parent { get; set; }
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

        public R92Header()
        {
            Init();
        }

        public R92Header(R92 parent)
        {
            Parent = parent;
            Init();
        }

        private void Init()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "IVT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = string.Empty, Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = "ACE", Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "R92", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Value = DateTime.Now, Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Value = DateTime.Now, Required = true };
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
    }

    public class R92Detail : IComparable<R92Detail>
    {
        public R92 Parent { get; set; }
        public int X85Id { get; set; }
        public X85 X85 { get; set; }
        public string CustomerCode { get; set; }
        public string PortCode { get; set; }
        public string SenderId { get; set; }
        public int X00Id { get; set; }
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> InvoiceNumber { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<DateTime> InvoiceDate { get; set; }
        public FixedPositionItem<string> GLAccountNumber { get; set; }
        public FixedPositionItem<string> DamageCode { get; set; }
        public FixedPositionItem<string> PIOCode { get; set; }
        public FixedPositionItem<string> OriginCode { get; set; }
        public FixedPositionItem<string> DestinationCode { get; set; }
        public FixedPositionItem<string> Filler1 { get; set; }
        public FixedPositionItem<char> CreditSign { get; set; }
        public FixedPositionItem<decimal> CreditAmount { get; set; }
        public FixedPositionItem<DateTime> CompletionDate { get; set; }
        public FixedPositionItem<string> Filler2 { get; set; }
        public FixedPositionItem<DateTime> CompletionTime { get; set; }
        public FixedPositionItem<string> Filler3 { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<bool> ManualInvoiceFlag { get; set; }
        public FixedPositionItem<string> Filler4 { get; set; }

        public R92Detail()
        {
            Init();
        }

        public R92Detail(R92 parent)
        {
            Parent = parent;
            Init();
        }

        private void Init()
        {
            CustomerCode = string.Empty; 
            PortCode = string.Empty;
            SenderId = string.Empty;
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "IVT01", Required = true };
            InvoiceNumber = new FixedPositionItem<string>() { Offset = 5, Length = 15, Value = string.Empty, Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 20, Length = 17, Value = string.Empty, Required = true };
            InvoiceDate = new FixedPositionItem<DateTime>() { Offset = 37, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            GLAccountNumber = new FixedPositionItem<string>() { Offset = 45, Length = 4, Value = string.Empty, Required = true };
            DamageCode = new FixedPositionItem<string>() { Offset = 49, Length = 6, Value = string.Empty };
            PIOCode = new FixedPositionItem<string>() { Offset = 55, Length = 4, Value = string.Empty };
            OriginCode = new FixedPositionItem<string>() { Offset = 59, Length = 7, Value = string.Empty, Required = true };
            DestinationCode = new FixedPositionItem<string>() { Offset = 66, Length = 7, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 73, Length = 1, Value = new string(Utils.FillerChar, 1) };
            CreditSign = new FixedPositionItem<char>() { Offset = 74, Length = 1, Value = '+', Required = true };
            CreditAmount = new FixedPositionItem<decimal>() { Offset = 75, Length = 8, Value = 0.0M, Format = "{0:00000000}", Required = true };
            CompletionDate = new FixedPositionItem<DateTime>() { Offset = 83, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            Filler2 = new FixedPositionItem<string>() { Offset = 91, Length = 1, Value = new string(Utils.FillerChar, 1) };
            CompletionTime = new FixedPositionItem<DateTime>() { Offset = 92, Format = "{0:HHmmss}", Length = 6 };
            Filler3 = new FixedPositionItem<string>() { Offset = 98, Length = 1, Value = new string(Utils.FillerChar, 1) };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 99, Length = 12, Value = string.Empty, Required = true };
            ManualInvoiceFlag = new FixedPositionItem<bool>() { Offset = 111, Length = 1, Value = false, Format = "{0:T;;F}", Required = true };
            Filler4 = new FixedPositionItem<string>() { Offset = 112, Length = 138, Value = new string(Utils.FillerChar, 138) };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                InvoiceNumber.ToString() +
                VIN.ToString() +
                InvoiceDate.ToString() +
                GLAccountNumber.ToString() +
                DamageCode.ToString() +
                PIOCode.ToString() +
                OriginCode.ToString() +
                DestinationCode.ToString() +
                Filler1.ToString() +
                CreditSign.ToString() +
                CreditAmount.ToString() +
                CompletionDate.ToString() +
                Filler2.ToString() +
                CompletionTime.ToString() +
                Filler3.ToString() +
                ShipmentAuthorizationCode.ToString() +
                ManualInvoiceFlag.ToString() +
                Filler4.ToString();
        }

        #region IComparable<R92Detail> Members

        public int CompareTo(R92Detail other)
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

    public class R92Detail_Comparer : IComparer<R92Detail>
    {
        #region IComparer<R92Detail> Members

        public int Compare(R92Detail x, R92Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class R92Trailer
    {
        public R92 Parent { get; set; }
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

        public R92Trailer()
        {
            Init();
        }

        public R92Trailer(R92 parent)
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
