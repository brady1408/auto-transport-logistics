using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    /// <summary>
    /// ACES Stock Transfer
    /// </summary>
    public class G70
    {
        public int X00Id { get; set; }
        public TransmissionInfo TransmissionInfo { get; set; }
        public DateTime CreatedDateTime { get; set; }
        private List<G70Detail> _detail = new List<G70Detail>();
        public G70Header Header { get; set; }
        public G70Trailer Trailer { get; set; }
        public List<G70Detail> Detail
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

        public G70()
        {
            CreatedDateTime = DateTime.Now;
            Header = new G70Header();
            Trailer = new G70Trailer();
        }

        public static G70 Load(TransmissionInfo ti)
        {
            return Load(ti, true);
        }

        public static G70 Load(TransmissionInfo ti, bool moveOnError)
        {
            if (!ti.LocalFile.Exists)
                return null;
            G70 rtn = new G70();
            rtn.TransmissionInfo = ti;

            string[] lines = File.ReadAllLines(ti.LocalFile.FullName);

            rtn.Detail = new List<G70Detail>();
            bool hasHdr = false, hasTrl = false;
            int detailCount = 0;
            bool movedToPending = false;

            foreach (string line in lines)
            {
                switch (line.Substring(0, 5))
                {
                    case "STT00":
                        if (hasHdr) continue;
                        hasHdr = true;
                        try
                        {
                            rtn.Header = G70Header.Load(line);
                        }
                        catch (FileValidationException fvEx)
                        {
                            Error err = new Error();
                            err.Message = fvEx.Message + " (Header)";
                            err.Description = "Missing required header information";
                            err.Code = "ACES_VALIDATION_EXCEPTION";
                            err.EdiSet = "G70";
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
                    case "STT01":
                        G70Detail d = null;
                        try
                        {
                            d = G70Detail.Load(line);
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
                            rtn.Trailer = G70Trailer.Load(line);
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
                            err.EdiSet = "G70";
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

            return false;
        }
    }

    public class G70Header
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

        public G70Header()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "STT00", Required = true };
            SenderID = new FixedPositionItem<string>() { Offset = 5, Length = 3, Value = "ACE", Required = true };
            ReceiverID = new FixedPositionItem<string>() { Offset = 8, Length = 3, Value = string.Empty, Required = true };
            TransmissionID = new FixedPositionItem<string>() { Offset = 11, Length = 3, Value = "G70", Required = true };
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

        public static G70Header Load(string headerLine)
        {
            if (headerLine.Equals(""))
                return null;
            G70Header rtn = new G70Header();

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

    public class G70Detail : IComparable<G70Detail>
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<string> VIN { get; set; }
        public FixedPositionItem<string> AuthorizationNumber { get; set; }
        public FixedPositionItem<DateTime> AuthorizedDate { get; set; }
        public FixedPositionItem<DateTime> RequiredPortReleaseDate { get; set; }
        public FixedPositionItem<string> DirNumOrigin { get; set; }
        public FixedPositionItem<string> DirNumDestination { get; set; }
        public FixedPositionItem<string> DestinationRampCode { get; set; }
        public FixedPositionItem<DateTime> EstimatedArrivalDate { get; set; }
        public FixedPositionItem<string> ExteriorColor { get; set; }
        public FixedPositionItem<string> InteriorColor { get; set; }
        public FixedPositionItem<string> KeyCode { get; set; }
        public FixedPositionItem<string> USModelCode { get; private set; }
        public FixedPositionItem<string> Filler { get; set; }
        public bool DoInsert { get; set; }

        public G70Detail()
        {
            RecordID = new FixedPositionItem<string>() { Offset = 0, Length = 5, Value = "STT01", Required = true };
            VIN = new FixedPositionItem<string>() { Offset = 5, Length = 17, Value = string.Empty, Required = true };
            AuthorizationNumber = new FixedPositionItem<string>() { Offset = 22, Length = 12, Value = string.Empty, Required = true };
            AuthorizedDate = new FixedPositionItem<DateTime>() { Offset = 34, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            RequiredPortReleaseDate = new FixedPositionItem<DateTime>() { Offset = 42, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            DirNumOrigin = new FixedPositionItem<string>() { Offset = 50, Length = 7, Value = string.Empty, Required = true };
            DirNumDestination = new FixedPositionItem<string>() { Offset = 57, Length = 7, Value = string.Empty, Required = true };
            DestinationRampCode = new FixedPositionItem<string>() { Offset = 64, Length = 5, Value = string.Empty, Required = true };
            EstimatedArrivalDate = new FixedPositionItem<DateTime>() { Offset = 69, Length = 8, Value = DateTime.Now, Format = "{0:yyyyMMdd}", Required = true };
            ExteriorColor = new FixedPositionItem<string>() { Offset = 77, Length = 3, Value = string.Empty, Required = true };
            InteriorColor = new FixedPositionItem<string>() { Offset = 80, Length = 3, Value = string.Empty, Required = true };
            KeyCode = new FixedPositionItem<string>() { Offset = 83, Length = 7, Value = string.Empty, Required = true };
            USModelCode = new FixedPositionItem<string>() { Offset = 90, Length = 8, Value = string.Empty, Required = true };
            Filler = new FixedPositionItem<string>() { Offset = 98, Length = 152, Value = string.Empty };
            DoInsert = false;
        }

        public override string ToString()
        {
            return
                RecordID.ToString() +
                VIN.ToString() +
                AuthorizationNumber.ToString() +
                AuthorizedDate.ToString() +
                RequiredPortReleaseDate.ToString() +
                DirNumOrigin.ToString() +
                DirNumDestination.ToString() +
                DestinationRampCode.ToString() +
                EstimatedArrivalDate.ToString() +
                ExteriorColor.ToString() +
                InteriorColor.ToString() +
                KeyCode.ToString() +
                USModelCode.ToString() +
                Filler.ToString();
        }

        public static G70Detail Load(string detailLine)
        {
            if (detailLine.Equals(""))
                return null;
            G70Detail rtn = new G70Detail();
            DateTime temp;

            rtn.RecordID.Value = detailLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            rtn.VIN.Value = detailLine.Substring(rtn.VIN.Offset, rtn.VIN.Length).Trim();
            rtn.AuthorizationNumber.Value = detailLine.Substring(rtn.AuthorizationNumber.Offset, rtn.AuthorizationNumber.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.AuthorizedDate.Offset, rtn.AuthorizedDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.AuthorizedDate.Value = temp;

            DateTime.TryParseExact(detailLine.Substring(rtn.RequiredPortReleaseDate.Offset, rtn.RequiredPortReleaseDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.RequiredPortReleaseDate.Value = temp;

            rtn.DirNumOrigin.Value = detailLine.Substring(rtn.DirNumOrigin.Offset, rtn.DirNumOrigin.Length).Trim();
            rtn.DirNumDestination.Value = detailLine.Substring(rtn.DirNumDestination.Offset, rtn.DirNumDestination.Length).Trim();
            rtn.DestinationRampCode.Value = detailLine.Substring(rtn.DestinationRampCode.Offset, rtn.DestinationRampCode.Length).Trim();

            DateTime.TryParseExact(detailLine.Substring(rtn.EstimatedArrivalDate.Offset, rtn.EstimatedArrivalDate.Length).Trim(),
                "yyyyMMdd", CultureInfo.InvariantCulture, DateTimeStyles.NoCurrentDateDefault, out temp);
            rtn.EstimatedArrivalDate.Value = temp;

            rtn.ExteriorColor.Value = detailLine.Substring(rtn.ExteriorColor.Offset, rtn.ExteriorColor.Length).Trim();
            rtn.InteriorColor.Value = detailLine.Substring(rtn.InteriorColor.Offset, rtn.InteriorColor.Length).Trim();
            rtn.KeyCode.Value = detailLine.Substring(rtn.KeyCode.Offset, rtn.KeyCode.Length).Trim();
            rtn.USModelCode.Value = detailLine.Substring(rtn.USModelCode.Offset, rtn.USModelCode.Length).Trim();

            return rtn;
        }

        #region IComparable<G70Detail> Members

        public int CompareTo(G70Detail other)
        {
            int pickupComp = this.VIN.Value.CompareTo(other.VIN.Value);
            int dropComp = this.DestinationRampCode.Value.CompareTo(other.DestinationRampCode.Value);

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

    public class G70Detail_Comparer : IComparer<G70Detail>
    {
        #region IComparer<G70Detail> Members

        public int Compare(G70Detail x, G70Detail y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }

    public class G70Trailer
    {
        public FixedPositionItem<string> RecordID { get; set; }
        public FixedPositionItem<int> TransmitRecordCount { get; set; }
        public FixedPositionItem<string> Filler { get; set; }

        public G70Trailer()
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

        public static G70Trailer Load(string trailerLine)
        {
            if (trailerLine.Equals(""))
                return null;
            G70Trailer rtn = new G70Trailer();

            rtn.RecordID.Value = trailerLine.Substring(rtn.RecordID.Offset, rtn.RecordID.Length).Trim();
            int trc;
            int.TryParse(trailerLine.Substring(rtn.TransmitRecordCount.Offset, rtn.TransmitRecordCount.Length).Trim(), out trc);
            rtn.TransmitRecordCount.Value = trc;

            return rtn;
        }
    }
}
