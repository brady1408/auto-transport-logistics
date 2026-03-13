using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Misc Move Authorization
    /// </summary>
    public class G95
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G95Detail> _detail = new List<G95Detail>();
        public G95Header Header { get; set; }
        public G95Trailer Trailer { get; set; }
        public List<G95Detail> Detail
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

        public G95()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G95Header();
            Trailer = new G95Trailer();
        }

        public static G95 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G95 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G95 rtn = new G95();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G95Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "MMT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G95Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G95";
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
                    case "MMT01":
                        G95Detail d = null;
                        try
                        {
                            d = G95Detail.Load(line);
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
                            rtn.Trailer = G95Trailer.Load(line);
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
                            err.EdiSet = "G95";
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
            return Process(db, true);
        }

        public bool Process(ATLDbDataContext db, bool doUpdate)
        {
            //bool result = true;
            DateTime creation = CreatedDateTime;

            Detail.Sort(new G95Detail_Comparer());
            List<G73Detail> used = new List<G73Detail>(Detail.Count);

            foreach (G95Detail d10 in Detail)
            {
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
                    d10.DoInsert = true;
                }
                else
                {
                    d10.DoUpdate = true;
                }
            }


            //while (Detail.Count > 0)
            for (int c = 0, j = 0; c < Detail.Count && j < Detail.Count; j++)
            {
                G95Detail det = Detail[c];
                List<G95Detail> order = new List<G95Detail>();
                order.Add(det);

                //if (!det.DropShipFlag.Value)
                //{
                for (int i = c + 1; i < Detail.Count; i++)
                {
                    if (det.CompareTo(Detail[i]) == 0)
                        order.Add(Detail[i]);
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
                try
                {
                    x01 = (from p in db.X01s
                           join o in db.X00s on p.X00Id equals o.X00Id
                           where o.DataFormatType.Equals("ACES") &&
                              (p.OriginCode.Trim().Equals(det.OriginCode.Value) ||
                              (det.OriginCode.Value.Trim().Equals("") &&
                              p.OLLocCode.Trim().Equals(det.OriginCode.Value))) &&
                              p.MfgCode.Trim().ToUpper().Equals(mcode.ToUpper())
                           select p).First() as X01;
                }
                catch (Exception ex)
                {
                    string addtl = string.Format("No valid X01 found. Origin Code/DestRailRampCode={0} OR OLLocCode/OriginCode={1}",
                        det.OriginCode.Value,
                        det.OriginCode.Value);
                    Error err = new Error();
                    err.Message = "No valid X01 found";
                    err.EdiSet = "G95";
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
                foreach (G95Detail d10 in order)
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
                                       p.DropCustNumber.Trim().ToUpper().Equals(det.DestinationCode.Value.ToUpper()) &&
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
                                det.DestinationCode.Value,
                                x01.BillCust,
                                mcode.ToUpper());

                            db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                        }
                        catch (Exception ex)
                        {
                            AppLog.WriteExceptionToLog(ex, null, true);
                            continue;
                        }

                        if (d00Id == null)
                        {
                            AppLog.WriteToLog("NULL D00ID (G95.cs Line 329)");
                        }

                        //if (det.DropShip.Value)
                        //{
                        //    try
                        //    {
                        //        D00 dsd00 = (from p in db.D00s
                        //                     where p.D00Id == d00Id
                        //                     select p).FirstOrDefault();

                        //        if (dsd00 != null)
                        //            dsd00.DropShip = (byte)1;

                        //        db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                        //    }
                        //    catch { }
                        //}
                    }
                }

                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                int count = 0;
                decimal totalPrice = 0M;
                if (doAppend)
                {
                    int x = 1;
                }
                foreach (G95Detail d10 in order)
                {
                    if (d10.DoInsert && d00Id != null)
                    {
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

                            insD10.AuthorizationCode = d10.ShipmentAuthorizationCode.Value;
                            insD10.SF1 = Header.TransmissionDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                            insD10.SF2 = d10.RequiredShipDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
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
                                     where p.VIN.Trim().Equals(d10.VIN.Value) &&
                                        creation.Equals(p.CreatedTimeString) &&
                                        p.CreatedBy.Trim().Equals("ACES")
                                     orderby p.D10Id descending
                                     select p).FirstOrDefault();

                            if (upD10 == null)
                                continue;

                            bool changesMade = false;

                            if (!upD10.AuthorizationCode.Equals(d10.ShipmentAuthorizationCode.Value))
                            {
                                upD10.AuthorizationCode = d10.ShipmentAuthorizationCode.Value;
                                changesMade = true;
                            }

                            if (upD10.SF1.Trim().Equals(""))
                            {
                                upD10.SF1 = Header.TransmissionDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
                                upD10.SF2 = d10.RequiredShipDate.Value.ToString("yyyy-MM-dd HH:mm:ss");
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
    }

    public class G95Header
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

        public G95Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "MMT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G95", Required = true };
            TransmissionDate = new FixedPositionItem<DateTime>() { Offset = 14, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            TransmissionTime = new FixedPositionItem<DateTime>() { Offset = 22, Length = 6, Format = "{0:HHmmss}", Required = true };
            PortCode = new FixedPositionItem<string>() { Offset = 28, Length = 2, Value = string.Empty };
            CustomerCode = new FixedPositionItem<string>() { Offset = 30, Length = 10, Value = string.Empty, Required = true };
            TotalRecordCount = new FixedPositionItem<int>() { Offset = 40, Length = 6, Value = 0, Format = "{0:000000}", Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 46, Length = 304, Value = new string(Utils.FillerChar, 204) };
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

        public static G95Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G95Header rtn = new G95Header();

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

    public class G95Detail : IComparable<G95Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<string> CarrierCode { get; set; }
        public FixedPositionItem<string> ModelCode { get; set; }
        public FixedPositionItem<string> ExteriorColor { get; set; }
        public FixedPositionItem<string> TransactionCode { get; set; }
        public FixedPositionItem<string> PrevVehicleOwnerName { get; set; }
        public FixedPositionItem<DateTime> TenderDate { get; set; }
        public FixedPositionItem<DateTime> RequiredShipDate { get; set; }
        public FixedPositionItem<DateTime> RequiredDeliveryDate { get; set; }
        public FixedPositionItem<decimal> Rate { get; set; }
        public FixedPositionItem<string> Comments { get; set; }
        public FixedPositionItem<string> AssignedTruckAccount { get; set; }
        public FixedPositionItem<DateTime> ApprovedDate { get; set; }
        public FixedPositionItem<string> ApproverName { get; set; }
        public FixedPositionItem<string> OriginCode { get; set; }
        public FixedPositionItem<string> OriginContactName { get; set; }
        public FixedPositionItem<string> OriginAddress1 { get; set; }
        public FixedPositionItem<string> OriginAddress2 { get; set; }
        public FixedPositionItem<string> OriginCity { get; set; }
        public FixedPositionItem<string> OriginState { get; set; }
        public FixedPositionItem<string> OriginZip { get; set; }
        public FixedPositionItem<string> OriginPhone { get; set; }
        public FixedPositionItem<string> OriginFax { get; set; }
        public FixedPositionItem<string> DestinationCode { get; set; }
        public FixedPositionItem<string> DestinationContactName { get; set; }
        public FixedPositionItem<string> DestinationAddress1 { get; set; }
        public FixedPositionItem<string> DestinationAddress2 { get; set; }
        public FixedPositionItem<string> DestinationCity { get; set; }
        public FixedPositionItem<string> DestinationState { get; set; }
        public FixedPositionItem<string> DestinationZip { get; set; }
        public FixedPositionItem<string> DestinationPhone { get; set; }
        public FixedPositionItem<string> DestinationFax { get; set; }
        public FixedPositionItem<string> Filler { get; set; }
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

        public G95Detail()
        { 
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "MMT01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 22, Length = 12, Value = string.Empty, Required = true };
            CarrierCode = new FixedPositionItem<string>() { Offset = 34, Length = 5, Value = string.Empty, Required = true };
            ModelCode = new FixedPositionItem<string>() { Offset = 39, Length = 8, Value = string.Empty, Required = true };
            ExteriorColor = new FixedPositionItem<string>() { Offset = 47, Length = 3, Value = string.Empty, Required = true };
            TransactionCode = new FixedPositionItem<string>() { Offset = 50, Length = 12, Value = string.Empty };
            PrevVehicleOwnerName = new FixedPositionItem<string>() { Offset = 62, Length = 20, Value = string.Empty };
            TenderDate = new FixedPositionItem<DateTime>() { Offset = 82, Length = 8, Format = "{0:yyyyMMdd}", Value = DateTime.Now };
            RequiredShipDate = new FixedPositionItem<DateTime>() { Offset = 90, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            RequiredDeliveryDate = new FixedPositionItem<DateTime>() { Offset = 98, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            Rate = new FixedPositionItem<decimal>() { Offset = 106, Length = 8, Value = 0M, Format = "{0:00000000}", Required = true };
            Comments = new FixedPositionItem<string>() { Offset = 114, Length = 40, Value = string.Empty };
            AssignedTruckAccount = new FixedPositionItem<string>() { Offset = 154, Length = 4, Value = string.Empty };
            ApprovedDate = new FixedPositionItem<DateTime>() { Offset = 158, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            ApproverName = new FixedPositionItem<string>() { Offset = 166, Length = 20, Value = string.Empty, Required = true };
            OriginCode = new FixedPositionItem<string>() { Offset = 186, Length = 7, Value = string.Empty, Required = true };
            OriginContactName = new FixedPositionItem<string>() { Offset = 193, Length = 20, Value = string.Empty, Required = true };
            OriginAddress1 = new FixedPositionItem<string>() { Offset = 213, Length = 30, Value = string.Empty, Required = true };
            OriginAddress2 = new FixedPositionItem<string>() { Offset = 243, Length = 30, Value = string.Empty, Required = true };
            OriginCity = new FixedPositionItem<string>() { Offset = 273, Length = 30, Value = string.Empty, Required = true };
            OriginState = new FixedPositionItem<string>() { Offset = 303, Length = 2, Value = string.Empty, Required = true };
            OriginZip = new FixedPositionItem<string>() { Offset = 305, Length = 10, Value = string.Empty, Required = true };
            OriginPhone = new FixedPositionItem<string>() { Offset = 315, Length = 20, Value = string.Empty, Required = true };
            OriginFax = new FixedPositionItem<string>() { Offset = 335, Length = 20, Value = string.Empty, Required = true };
            DestinationCode = new FixedPositionItem<string>() { Offset = 355, Length = 7, Value = string.Empty, Required = true };
            DestinationContactName = new FixedPositionItem<string>() { Offset = 362, Length = 20, Value = string.Empty, Required = true };
            DestinationAddress1 = new FixedPositionItem<string>() { Offset = 382, Length = 30, Value = string.Empty, Required = true };
            DestinationAddress2 = new FixedPositionItem<string>() { Offset = 412, Length = 30, Value = string.Empty, Required = true };
            DestinationCity = new FixedPositionItem<string>() { Offset = 442, Length = 30, Value = string.Empty, Required = true };
            DestinationState = new FixedPositionItem<string>() { Offset = 472, Length = 2, Value = string.Empty, Required = true };
            DestinationZip = new FixedPositionItem<string>() { Offset = 474, Length = 10, Value = string.Empty, Required = true };
            DestinationPhone = new FixedPositionItem<string>() { Offset = 484, Length = 20, Value = string.Empty, Required = true };
            DestinationFax = new FixedPositionItem<string>() { Offset = 504, Length = 20, Value = string.Empty, Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 524, Length = 26, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                ShipmentAuthorizationCode.ToString() +
                CarrierCode.ToString() +
                ModelCode.ToString() +
                ExteriorColor.ToString() +
                TransactionCode.ToString() +
                PrevVehicleOwnerName.ToString() +
                TenderDate.ToString() +
                RequiredShipDate.ToString() +
                RequiredDeliveryDate.ToString() +
                Rate.ToString() +
                Comments.ToString() +
                AssignedTruckAccount.ToString() +
                ApprovedDate.ToString() +
                ApproverName.ToString() +
                OriginCode.ToString() +
                OriginContactName.ToString() +
                OriginAddress1.ToString() +
                OriginAddress2.ToString() +
                OriginCity.ToString() +
                OriginState.ToString() +
                OriginZip.ToString() +
                OriginPhone.ToString() +
                OriginFax.ToString() +
                DestinationCode.ToString() +
                DestinationContactName.ToString() +
                DestinationAddress1.ToString() +
                DestinationAddress2.ToString() +
                DestinationCity.ToString() +
                DestinationState.ToString() +
                DestinationZip.ToString() +
                DestinationPhone.ToString() +
                DestinationFax.ToString() +
                Filler.ToString();
        }

        public static G95Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G95Detail rtn = new G95Detail();

            DateTime temp;
            decimal rate;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.ShipmentAuthorizationCode.Value = detailLine.Substring(rtn.ShipmentAuthorizationCode.Offset, 
                rtn.ShipmentAuthorizationCode.Length).Trim();
            rtn.CarrierCode.Value = detailLine.Substring(rtn.CarrierCode.Offset, rtn.CarrierCode.Length).Trim();
            rtn.ModelCode.Value = detailLine.Substring(rtn.ModelCode.Offset, rtn.ModelCode.Length).Trim();
            rtn.ExteriorColor.Value = detailLine.Substring(rtn.ExteriorColor.Offset, rtn.ExteriorColor.Length).Trim();
            rtn.TransactionCode.Value = detailLine.Substring(rtn.TransactionCode.Offset, rtn.TransactionCode.Length).Trim();
            rtn.PrevVehicleOwnerName.Value = detailLine.Substring(rtn.PrevVehicleOwnerName.Offset, 
                rtn.PrevVehicleOwnerName.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.TenderDate.Offset, rtn.TenderDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.TenderDate.Value = temp;

            DateTime.TryParseExact(detailLine.Substring(rtn.RequiredShipDate.Offset, rtn.RequiredShipDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.RequiredShipDate.Value = temp;

            DateTime.TryParseExact(detailLine.Substring(rtn.RequiredDeliveryDate.Offset, rtn.RequiredDeliveryDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.RequiredDeliveryDate.Value = temp;

            decimal.TryParse(detailLine.Substring(rtn.Rate.Offset, rtn.Rate.Length).Trim(), out rate);
            rtn.Rate.Value = rate / 100;

            rtn.Comments.Value = detailLine.Substring(rtn.Comments.Offset, rtn.Comments.Length).Trim();
            rtn.AssignedTruckAccount.Value = detailLine.Substring(rtn.AssignedTruckAccount.Offset, 
                rtn.AssignedTruckAccount.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.ApprovedDate.Offset, rtn.ApprovedDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.ApprovedDate.Value = temp;

            rtn.ApproverName.Value = detailLine.Substring(rtn.ApproverName.Offset, rtn.ApproverName.Length).Trim();
            rtn.OriginCode.Value = detailLine.Substring(rtn.OriginCode.Offset, rtn.OriginCode.Length).Trim();
            rtn.OriginContactName.Value = detailLine.Substring(rtn.OriginContactName.Offset, 
                rtn.OriginContactName.Length).Trim();
            rtn.OriginAddress1.Value = detailLine.Substring(rtn.OriginAddress1.Offset, rtn.OriginAddress1.Length).Trim();
            rtn.OriginAddress2.Value = detailLine.Substring(rtn.OriginAddress2.Offset, rtn.OriginAddress2.Length).Trim();
            rtn.OriginCity.Value = detailLine.Substring(rtn.OriginCity.Offset, rtn.OriginCity.Length).Trim();
            rtn.OriginState.Value = detailLine.Substring(rtn.OriginState.Offset, rtn.OriginState.Length).Trim();
            rtn.OriginZip.Value = detailLine.Substring(rtn.OriginZip.Offset, rtn.OriginZip.Length).Trim();
            rtn.OriginPhone.Value = detailLine.Substring(rtn.OriginPhone.Offset, rtn.OriginPhone.Length).Trim();
            rtn.OriginFax.Value = detailLine.Substring(rtn.OriginFax.Offset, rtn.OriginFax.Length).Trim();
            rtn.DestinationCode.Value = detailLine.Substring(rtn.DestinationCode.Offset, rtn.DestinationCode.Length).Trim();
            rtn.DestinationContactName.Value = detailLine.Substring(rtn.DestinationContactName.Offset, 
                rtn.DestinationContactName.Length).Trim();
            rtn.DestinationAddress1.Value = detailLine.Substring(rtn.DestinationAddress1.Offset, 
                rtn.DestinationAddress1.Length).Trim();
            rtn.DestinationAddress2.Value = detailLine.Substring(rtn.DestinationAddress2.Offset, 
                rtn.DestinationAddress2.Length).Trim();
            rtn.DestinationCity.Value = detailLine.Substring(rtn.DestinationCity.Offset, rtn.DestinationCity.Length).Trim();
            rtn.DestinationState.Value = detailLine.Substring(rtn.DestinationState.Offset, rtn.DestinationState.Length).Trim();
            rtn.DestinationZip.Value = detailLine.Substring(rtn.DestinationZip.Offset, rtn.DestinationZip.Length).Trim();
            rtn.DestinationPhone.Value = detailLine.Substring(rtn.DestinationPhone.Offset, rtn.DestinationPhone.Length).Trim();
            rtn.DestinationFax.Value = detailLine.Substring(rtn.DestinationFax.Offset, rtn.DestinationFax.Length).Trim();

            return rtn;
        }

        #region IComparable<G95Detail> Members

        public int CompareTo(G95Detail other)
        {
            int pickupComp = this.VIN.Value.CompareTo(other.VIN.Value);
            int dropComp = this.ShipmentAuthorizationCode.Value.CompareTo(other.ShipmentAuthorizationCode.Value);

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

    public class G95Detail_Comparer : IComparer<G95Detail>
    {
        #region IComparer<G95Detail> Members

        public int Compare(G95Detail x, G95Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G95Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G95Trailer()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = Utils.EOF, Required = true };
            TransmitRecordCount = new FixedPositionItem<int>() { Offset = 5, Length = 6, Value = 0, Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 11, Length = 339, Value = string.Empty };
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                TransmitRecordCount.ToString() +
                Filler.ToString();
        }

        public static G95Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G95Trailer rtn = new G95Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
