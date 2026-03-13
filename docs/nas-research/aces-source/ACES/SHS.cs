using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.ComponentModel;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Conversion Data Format
    /// </summary>
    public class SHS
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        public int RecordCount
        {
            get { return Detail.Count; }
        }
        public List<SHSDetail> Detail { get; set; }

        public SHS()
        {
            CreatedDateTime = DateTime.Now;
            Detail = new List<SHSDetail>();
        }

        public static SHS Generate(ATLDbDataContext db, BackgroundWorker worker, DateTime time)
        {
            SHS rtn = new SHS();
            rtn.CreatedDateTime = time;

            var q = from p in db.vw_edi_aces_conversions
                    select p;

            int x00Id = -1;

            int total = q.Count();
            int count = 0;
            if (total < 1)
                return null;

            ProcessProgress pp = new ProcessProgress();
            pp.State = OperationState.Query;

            foreach (vw_edi_aces_conversion r in q)
            {
                SHSDetail d = new SHSDetail();

                try
                {
                    d.X85 = (from p in db.X85s
                             where p.X85Id == r.X85ID
                             select p).FirstOrDefault() as X85;

                    d.X85.Processed = 1;
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }

                d.BillOfLadingNum.Value = r.bill_of_lading.ToString();
                d.DamageIndicator.Value = false;
                d.DeliveryStatusCode.Value = r.delivery_status_code;
                d.DestinationCode.Value = r.destination_code;
                d.ShipmentAuthorizationCode.Value = r.shipment_authorization_code;
                d.SPLCTransmissionFlag.Value = false;
                try
                {
                    d.StatusDate.Value = DateTime.ParseExact(r.status_date.Trim(), "yyyyMMdd", CultureInfo.InvariantCulture);
                    d.StatusTime.Value = DateTime.ParseExact(r.status_time.Trim(), "HHmmss", CultureInfo.InvariantCulture);
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }

                try
                {
                    d.LastUpdateDateTime.Value = DateTime.ParseExact(r.last_updated_datetime.Trim(),
                        "yyyyMMddHHmmss", CultureInfo.InvariantCulture);
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                    d.LastUpdateDateTime.Value = DateTime.Now;
                }
                d.PortOrAARRampCode.Value = r.ramp_code;
                d.VIN.Value = r.VIN;
                d.X85Id = r.X85ID;

                rtn.Detail.Add(d);

                count++;
                pp.StatusMessage = string.Format("Generating conversion data ({0} of {1})...",
                    count, total);
                worker.ReportProgress(Utils.CalculatePercentage(count, total), pp);
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
            }

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

                foreach (SHSDetail d in Detail)
                    sw.WriteLine(d);
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }
            finally
            {
                sw.Close();
            }

        }

        public string GetFileName(int dicn)
        {
            return string.Format("{0}_{1:000}_{2:000}.txt",
                "SHS",
                dicn,
                RecordCount);
        }
    }

   
    public class SHSDetail
    {
        public X85 X85 { get; set; }
        public int X85Id { get; set; }
        public FixedPositionItem<string> BillOfLadingNum { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<DateTime> StatusDate { get; set; }
        public FixedPositionItem<DateTime> StatusTime { get; set; }
        public FixedPositionItem<string> DeliveryStatusCode { get; set; }
        public FixedPositionItem<string> SPLCCode { get; set; }
        public FixedPositionItem<string> PortOrAARRampCode { get; set; }
        public FixedPositionItem<string> DestinationCode { get; set; }
        public FixedPositionItem<char> TruckType { get; set; }
        public FixedPositionItem<bool> DamageIndicator { get; set; }
        public FixedPositionItem<string> ShipmentAuthorizationCode { get; set; }
        public FixedPositionItem<bool> SPLCTransmissionFlag { get; set; }
        public FixedPositionItem<DateTime> LastUpdateDateTime { get; set; }

        public SHSDetail()
        {
            BillOfLadingNum = new FixedPositionItem<string>() { Offset = 0, Length = 15, Value = "", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 15, Length = 17, Value = "", Required = true };
            StatusDate = new FixedPositionItem<DateTime>() { Offset = 32, Length = 8, Format = "{0:yyyyMMdd}", Required = true };
            StatusTime = new FixedPositionItem<DateTime>() { Offset = 40, Length = 6, Format = "{0:HHmmss}", Required = true };
            DeliveryStatusCode = new FixedPositionItem<string>() { Offset = 46, Length = 3, Value = "", Required = true };
            SPLCCode = new FixedPositionItem<string>() { Offset = 49, Length = 10, Value = "" };
            PortOrAARRampCode = new FixedPositionItem<string>() { Offset = 59, Length = 5, Value = "", Required = true };
            DestinationCode = new FixedPositionItem<string>() { Offset = 64, Length = 7, Value = "", Required = true };
            TruckType = new FixedPositionItem<char>() { Offset = 64, Length = 1, Value = ' ', Required = true };
            DamageIndicator = new FixedPositionItem<bool>() { Offset = 72, Length = 1, Format = "{0:Y;;N}", Value = false, Required = true };
            ShipmentAuthorizationCode = new FixedPositionItem<string>() { Offset = 73, Length = 12, Value = "" };
            SPLCTransmissionFlag = new FixedPositionItem<bool>() { Offset = 85, Length = 1, Value = false, Format = "{0:T;;F}" };
            LastUpdateDateTime = new FixedPositionItem<DateTime>() { Offset = 86, Length = 14, Format="{0:yyyyMMddHHmmss}", Required = true };
        }

        public override string ToString()
        {
            return
                BillOfLadingNum.ToString() + 
                VIN.ToString() + 
                StatusDate.ToString() + 
                StatusTime.ToString() + 
                DeliveryStatusCode.ToString() + 
                SPLCCode.ToString() + 
                PortOrAARRampCode.ToString() + 
                DestinationCode.ToString() + 
                TruckType.ToString() + 
                DamageIndicator.ToString() + 
                ShipmentAuthorizationCode.ToString() +
                SPLCTransmissionFlag.ToString() + 
                LastUpdateDateTime.ToString();
        }
    }

}
