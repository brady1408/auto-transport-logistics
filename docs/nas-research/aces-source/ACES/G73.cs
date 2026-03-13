using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES VIN Allocation (ASN)
    /// </summary>
    public class G73
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G73Detail> _detail = new List<G73Detail>();
        public G73Header Header { get; set; }
        public G73Trailer Trailer { get; set; }
        public List<G73Detail> Detail
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

        public G73()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G73Header();
            Trailer = new G73Trailer();
        }

        public static G73 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G73 Load(TransmissionInfo ti, bool moveOnError)
        {
            if(!ti.LocalFile.Exists)
                return null;
            G73 rtn = new G73();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G73Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;
            bool hadDetailErrs = false;
            string detailErrs = rtn.TransmissionInfo.LocalFile.Name + Environment.NewLine + Environment.NewLine;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "VAT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G73Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G73";
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
                    case "VAT01":
                        G73Detail d = null;
                        try
                        {
                            d = G73Detail.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            hadDetailErrs = true;
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Detail)";
                            err.Description = "Missing required information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G73";
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
                            rtn.Trailer = G73Trailer.Load(line);
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
                            err.EdiSet = "G73";
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

            //if (hadDetailErrs)
            //{
            //    Utils.SendEmail("ACES Incoming File Validation Exception", detailErrs);
            //}

            return rtn;
        }

        public bool Process(ATLDbDataContext db)
        {
            return Process(db, true);
        }

        public bool Process(ATLDbDataContext db, bool doUpdate)
        {
            //bool result = true;
            DateTime creation = CreatedDateTime;

            Detail.Sort(new G73Detail_Comparer());
            List<G73Detail> used = new List<G73Detail>(Detail.Count);

            foreach (G73Detail d10 in Detail)
            {
                Console.Write("Checking database for VIN: {0}...", d10.VIN);
                int vinCount = 0;
                try
                {
                    vinCount = (from p in db.D10s
                                where p.VIN.Trim().Equals(d10.VIN) &&
                                !p.Status.Trim().Equals("Canceled")
                                select p).Count();
                }
                catch (Exception ex)
                {
                    //NOTE: No need to print to log; an exception here is normal
                    //AppLog.WriteExceptionToLog(ex, null, true);
                }

                if (vinCount < 1)
                {
                    Console.WriteLine("No VIN found!");
                    d10.DoInsert = true;
                }
                else
                {
                    Console.WriteLine("VIN found.");
                    d10.DoUpdate = true;
                }
            }

            //while (Detail.Count > 0)
            for (int c = 0, j = 0; c < Detail.Count && j < Detail.Count; j++ )
            {
                G73Detail det = Detail[c];
                List<G73Detail> order = new List<G73Detail>();
                order.Add(det);
                Console.WriteLine("Creating order from {0} to {1}", det.DestinationRailRampCode, det.TruckDestinationCode);
                //if (!det.DropShipFlag.Value)
                //{
                    for (int i = c + 1; i < Detail.Count; i++)
                    {
                        if (det.CompareTo(Detail[i]) == 0)
                        {
                            order.Add(Detail[i]);
                            Console.WriteLine("\tAdding VIN {0}", Detail[i].VIN);
                        }
                        else
                            break;
                    }
                //}

                c += order.Count;

                //insert order
                X01 x01 = null;
                string mcode = "";
                switch (Header.CustomerCode.Value.Trim().ToUpper())
                {
                    case "HMA":
                        mcode = "Hyundai";
                        break;
                    case "KMA":
                        mcode = "Kia";
                        break;
                    //case "VW":
                    //    mcode = "VW";
                    //    break;
                }
                Console.WriteLine("Querying X01 for ACES-{0}", mcode);
                try
                {
                    x01 = (from p in db.X01s
                           join o in db.X00s on p.X00Id equals o.X00Id
                           where o.DataFormatType.Equals("ACES") &&
                              (p.OriginCode.Trim().Equals(det.DestinationRailRampCode.Value) ||
                              (det.DestinationRailRampCode.Value.Trim().Equals("") &&
                              p.OLLocCode.Trim().Equals(det.OriginCode.Value))) &&
                              p.MfgCode.Trim().ToUpper().Equals(mcode.ToUpper())
                           select p).First() as X01;
                }
                catch (Exception ex)
                {
                    Console.WriteLine("ERR: No X01 Found!!");
                    string addtl = string.Format("No valid X01 found. Origin Code/DestRailRampCode={0} OR OLLocCode/OriginCode={1}", 
                        det.DestinationRailRampCode.Value,
                        det.OriginCode.Value);
                    Error err = new Error();
                    err.Message = "No valid X01 found";
                    err.EdiSet = "G73";
                    err.VIN = det.VIN.Value;
                    err.ErrorDateTime = creation;
                    err.Active = true;
                    err.Description = addtl;
                    err.System = "ACES";
                    err.Code = "ACES_CONFIGURATION_ERROR";
                    Utils.AddErrorEntry(err);
                    //AppLog.WriteExceptionToLog(ex, addtl, true);
                    continue;
                }

                int numInsert = 0;
                bool updating = false;
                foreach (G73Detail d10 in order)
                {
                    if (d10.DoInsert)
                        numInsert++;
                    else if (d10.DoUpdate)
                        updating = true;
                }

                if (numInsert < 1 && (!updating))
                    continue;

                bool doAppend = false;
                bool doSubmitChanges = false;
                int? d00Id = null;
                if (numInsert > 0)
                {
                    D00 appendOrder = null;
                    try
                    {
                        appendOrder = (from p in db.D00s
                                       where p.LoadCustNumber.ToUpper().Equals(x01.LoadCust.Trim().ToUpper()) &&
                                       p.DropCustNumber.Trim().ToUpper().Equals(det.TruckDestinationCode.Value.ToUpper()) &&
                                       p.MfgCode.ToUpper().Equals(x01.MfgCode.ToUpper()) &&
                                       p.OrderDate != null
                                       orderby p.OrderDate descending
                                       select p).FirstOrDefault() as D00;
                    }
                    catch { }

                    int appendDays = x01.OrderAppendDays ?? 0;
                    if (appendOrder != null && appendOrder.OrderDate.Value.AddDays(appendDays) >= CreatedDateTime)
                    {
                        doAppend = true;
                        d00Id = appendOrder.D00Id;
                        //appendOrder.UnitCount += numInsert;
                        db.SubmitChanges();
                    }

                    if (!doAppend)
                    {
                        Console.Write("Creating Order...");
                        try
                        {
                            db.sp_edi_D00_Insert(
                                ref d00Id,
                                Header.TransmissionDate.Value,
                                numInsert,
                                x01.MfgCode,
                                (byte)1,
                                (byte)1,
                                "ACES",
                                DateTime.Now,
                                x01.LocationCode,
                                x01.DispatchCode,
                                x01.OLLocCode,
                                x01.LoadCust,
                                det.TruckDestinationCode.Value,
                                x01.BillCust,
                                mcode.ToUpper());

                            db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                        }
                        catch (Exception ex)
                        {
                            Console.WriteLine("Failure.");
                            AppLog.WriteExceptionToLog(ex, null, true);
                            continue;
                        }

                        if (d00Id == null)
                        {
                            AppLog.WriteToLog("NULL D00ID (G73.cs Line 175)");
                        }

                        if (det.DropShipFlag.Value)
                        {
                            try
                            {
                                D00 dsd00 = (from p in db.D00s
                                             where p.D00Id == d00Id
                                             select p).FirstOrDefault();

                                if (dsd00 != null)
                                    dsd00.DropShip = (byte)1;

                                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                            }
                            catch { }
                        }
                    }
                }

                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                int count = 0;
                decimal totalPrice = 0M;
                if (doAppend)
                {
                    int x = 1;
                }
                foreach (G73Detail d10 in order)
                {
                    if (d10.DoInsert && d00Id != null)
                    {
                        Console.WriteLine("\tAdding VIN {0} to Order #{1}", d10.VIN, d00Id);
                        try
                        {
                            db.sp_edi_d10_insert(
                                d00Id,
                                (++count).ToString(),
                                null,
                                d10.VIN.Value,
                                "Inbound",
                                "ACES",
                                creation,
                                null,
                                null);

                            db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                        }
                        catch (Exception ex)
                        {
                            count--;
                            AppLog.WriteExceptionToLog(ex, null, true);
                            continue;
                        }

                        D10 insD10 = null;
                        try
                        {
                            insD10 = (from p in db.D10s
                                     where p.VIN.Trim().Equals(d10.VIN.Value) &&
                                        creation.Equals(p.CreatedTimeString) && 
                                        p.CreatedBy.Trim().Equals("ACES")
                                     orderby p.D10Id descending
                                     select p).FirstOrDefault();
                            if (insD10 == null)
                                continue;

                            insD10.AuthorizationCode = d10.ShipAuthorizationCode.Value;
                            insD10.SF1 = Header.TransmissionDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                            insD10.SF2 = d10.RequiredPortReleaseDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                            //insD10.DestRouteCode = d10.RouteCode.Value;
                            doSubmitChanges = true;
                        }
                        catch (Exception ex)
                        {
                            AppLog.WriteExceptionToLog(ex, string.Format("Unable to retrieve D10 for VIN: {0}", d10.VIN.Value), true);
                            continue;
                        }

                        Utils.DecodeAndPriceD10(insD10, db, ref totalPrice);
                    }
                    else if (d10.DoUpdate && doUpdate)
                    {
                        D10 upD10 = null;
                        try
                        {
                            upD10 = (from p in db.D10s
                                     where p.VIN.Trim().Equals(d10.VIN.Value) //&&
                                        //creation.Equals(p.CreatedTimeString) &&
                                        //p.CreatedBy.Trim().Equals("ACES")
                                     orderby p.D10Id descending
                                     select p).FirstOrDefault();

                            if (upD10 == null)
                                continue;

                            bool changesMade = false;

                            if (!upD10.AuthorizationCode.Equals(d10.ShipAuthorizationCode.Value))
                            {
                                upD10.AuthorizationCode = d10.ShipAuthorizationCode.Value;
                                changesMade = true;
                            }

                            if (upD10.SF1.Trim().Equals(""))
                            {
                                upD10.SF1 = Header.TransmissionDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                                upD10.SF2 = d10.RequiredPortReleaseDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                                changesMade = true;
                            }

                            if (changesMade)
                            {
                                upD10.UpdatedBy = "ACES";
                                upD10.UpdatedTimeString = CreatedDateTime;
                                doSubmitChanges = true;
                            }

                            //D00 upD00 = (from p in db.D00s
                            //             where p.D00Id == upD10.D00Id.Value
                            //             orderby p.D00Id descending
                            //             select p).FirstOrDefault();

                            //if (upD00 == null) continue;


                        }
                        catch (Exception ex)
                        {
                            AppLog.WriteExceptionToLog(ex, string.Format("Unable to update D10 for VIN: {0}", d10.VIN.Value), true);
                            continue;
                        }
                    }

                    if (doSubmitChanges)
                        db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                }

                if (totalPrice > 0M)
                {
                    try
                    {
                        D00 d00 = (from p in db.D00s
                                   where p.D00Id == d00Id
                                   select p).FirstOrDefault() as D00;

                        if (d00 != null)
                        {
                            d00.TotalAmount = totalPrice;
                            db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                        }
                    }
                    catch (Exception ex)
                    {
                        AppLog.WriteExceptionToLog(ex, string.Format("Unable to update order total for D00Id: {0}", d00Id), true);
                    }
                }
            }

            return true;
        }

        public string GetFileName(int dicn)
        {
            return string.Format("{0}{1:0000000}.txt",
                "G73",
                dicn);
        }
    }

    public class G73Header
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

        public G73Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "VAT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G73", Required = true };
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

        public static G73Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G73Header rtn = new G73Header();

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

    public class G73Detail : IComparable<G73Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> TruckDestinationCode { get; set; }
        public FixedPositionItem<bool> DropShipFlag { get; set; }
        public FixedPositionItem<string> RailCarrierExOrigin { get; set; }
        public FixedPositionItem<string> DestinationRailRampCode { get; set; }
        public FixedPositionItem<string> TruckVendorCode { get; set; }
        public FixedPositionItem<DateTime> RequiredPortReleaseDate { get; set; }
        public FixedPositionItem<string> ExteriorColorCode { get; set; }
        public FixedPositionItem<string> OriginCode { get; set; }
        public FixedPositionItem<string> ShipAuthorizationCode { get; set; }
        public FixedPositionItem<string> RouteCode { get; set; }
        public FixedPositionItem<string> Filler1 { get; private set; }
        public FixedPositionItem<int> AssignedTruckAcctNum { get; set; }
        public FixedPositionItem<string> Filler2 { get; private set; }
        private bool _doInsert = false;
        private bool _doUpdate = false;
        public bool DoInsert
        {
            get { return _doInsert; }
            set
            {
                _doInsert = value;
                if (value && _doUpdate)
                    _doUpdate = false;
            }
        }
        public bool DoUpdate
        {
            get { return _doUpdate; }
            set
            {
                _doUpdate = value;
                if (value && _doInsert)
                    _doInsert = false;
            }
        }

        public G73Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "VAT01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            TruckDestinationCode = new FixedPositionItem<string>() { Offset = 22, Length = 7, Value = string.Empty, Required = true };
            DropShipFlag = new FixedPositionItem<bool>() { Offset = 29, Length = 1, Value = false, Format="{0:Y;;N}", Required = true };
            RailCarrierExOrigin = new FixedPositionItem<string>() { Offset = 30, Length = 5, Value = string.Empty };
            DestinationRailRampCode = new FixedPositionItem<string>() { Offset = 35, Length = 5, Value = string.Empty };
            TruckVendorCode = new FixedPositionItem<string>() { Offset = 40, Length = 5, Value = string.Empty, Required = true };
            RequiredPortReleaseDate = new FixedPositionItem<DateTime>() { Offset = 45, Length = 8, Format="{0:yyyyMMdd}", Required = true };
            ExteriorColorCode = new FixedPositionItem<string>() { Offset = 53, Length = 3, Value = string.Empty, Required = true };
            OriginCode = new FixedPositionItem<string>() { Offset = 56, Length = 2, Value = string.Empty, Required = true };
            ShipAuthorizationCode = new FixedPositionItem<string>() { Offset = 58, Length = 12, Value = string.Empty, Required = true };
            RouteCode = new FixedPositionItem<string>() { Offset = 70, Length = 20, Value = string.Empty, Required = true };
            Filler1 = new FixedPositionItem<string>() { Offset = 90, Length = 1, Value = new string(Utils.FillerChar, 1) };
            AssignedTruckAcctNum = new FixedPositionItem<int>() { Offset = 91, Length = 4 };
            Filler2 = new FixedPositionItem<string>() { Offset = 95, Length = 155, Value = new string(Utils.FillerChar, 155) };
            DoInsert = false;
            DoUpdate = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                TruckDestinationCode.ToString() +
                DropShipFlag.ToString() +
                RailCarrierExOrigin.ToString() +
                DestinationRailRampCode.ToString() +
                TruckVendorCode.ToString() +
                RequiredPortReleaseDate.ToString() +
                ExteriorColorCode.ToString() +
                OriginCode.ToString() +
                ShipAuthorizationCode.ToString() +
                RouteCode.ToString() +
                Filler1.ToString() +
                AssignedTruckAcctNum.ToString() +
                Filler2.ToString();
        }

        public static G73Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G73Detail rtn = new G73Detail();

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.TruckDestinationCode.Value = detailLine.Substring(rtn.TruckDestinationCode.Offset, rtn.TruckDestinationCode.Length).Trim();
            
            string boolTest = detailLine.Substring(rtn.DropShipFlag.Offset, rtn.DropShipFlag.Length).Trim();
            rtn.DropShipFlag.Value = boolTest.Equals("Y") || boolTest.Equals("T") ? true : false;

            rtn.RailCarrierExOrigin.Value = detailLine.Substring(rtn.RailCarrierExOrigin.Offset, rtn.RailCarrierExOrigin.Length).Trim();
            rtn.DestinationRailRampCode.Value = detailLine.Substring(rtn.DestinationRailRampCode.Offset, rtn.DestinationRailRampCode.Length).Trim();
            rtn.TruckVendorCode.Value = detailLine.Substring(rtn.TruckVendorCode.Offset, rtn.TruckVendorCode.Length).Trim();

            string portReleaseDateTimeString = detailLine.Substring(rtn.RequiredPortReleaseDate.Offset, rtn.RequiredPortReleaseDate.Length).Trim();
            rtn.RequiredPortReleaseDate.Value = DateTime.ParseExact(portReleaseDateTimeString, "yyyyMMdd", CultureInfo.InvariantCulture);
            rtn.ExteriorColorCode.Value = detailLine.Substring(rtn.ExteriorColorCode.Offset, rtn.ExteriorColorCode.Length).Trim();
            rtn.OriginCode.Value = detailLine.Substring(rtn.OriginCode.Offset, rtn.OriginCode.Length).Trim();
            rtn.ShipAuthorizationCode.Value = detailLine.Substring(rtn.ShipAuthorizationCode.Offset, rtn.ShipAuthorizationCode.Length).Trim();
            rtn.RouteCode.Value = detailLine.Substring(rtn.RouteCode.Offset, rtn.RouteCode.Length).Trim();
            int trc;
            int.TryParse(detailLine.Substring(rtn.AssignedTruckAcctNum.Offset, rtn.AssignedTruckAcctNum.Length).Trim(), out trc);
            rtn.AssignedTruckAcctNum.Value = trc;

            return rtn;
        }

        #region IComparable<G73Detail> Members

        public int CompareTo(G73Detail other)
        {
            int pickupComp = this.DestinationRailRampCode.Value.CompareTo(other.DestinationRailRampCode.Value);
            int dropComp = this.TruckDestinationCode.Value.CompareTo(other.TruckDestinationCode.Value);

            if (pickupComp == 0 && dropComp == 0)
            {
                if (this.DropShipFlag.Value == other.DropShipFlag.Value)
                    return 0;
                else
                    return -1;
            }
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
                else if (this.DropShipFlag.Value == other.DropShipFlag.Value)
                    return 0;
                else
                    return -1;

            }
        }

        #endregion
    }

    public class G73Detail_Comparer : IComparer<G73Detail>
    {
        #region IComparer<G73Detail> Members

        public int Compare(G73Detail x, G73Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G73Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G73Trailer()
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

        public static G73Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G73Trailer rtn = new G73Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
