using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Data.SqlClient;
using System.IO;
using System.Windows.Forms;
using System.Net.Mail;
using System.Net;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    public static class Utils
    {
        public const char FillerChar = ' ';
        public const string EOF = "EOF  ";
        private static bool didResetDICN = false;
        private static List<Error> errors = new List<Error>();
        private static string errorEmailAddr = "dev@atlinksystem.com";
        public static bool checkErrorTable = true;

        public static void LogError(ATLDbDataContext db, Error err, string connStr)
        {
            CheckErrorTableExists(db, connStr);

            List<Error> dbErrs = (from p in db.Errors
                                  where p.Active == true &&
                                  p.System.ToUpper().Equals("ACES")
                                  select p).ToList();

            DBErrorEqualityComparer comparer = new DBErrorEqualityComparer();
            if (dbErrs.Contains(err, comparer))
            {
                db.Errors.InsertOnSubmit(err);
                db.SubmitChanges();
            }
        }

        public static void LogAllErrors(ATLDbDataContext db, string connStr)
        {
            if (errors.Count < 1) return;
            CheckErrorTableExists(db, connStr);

            List<Error> dbErrs = (from p in db.Errors
                                  where p.Active == true &&
                                  p.System.ToUpper().Equals("ACES")
                                  select p).ToList();

            List<Error> toInsert = new List<Error>();
            DBErrorEqualityComparer comparer = new DBErrorEqualityComparer();
            foreach (Error e in errors)
            {
                if (!dbErrs.Contains(e, comparer))
                    toInsert.Add(e);
            }

            db.Errors.InsertAllOnSubmit(toInsert);
            db.SubmitChanges();  
        }

        internal class DBErrorEqualityComparer : IEqualityComparer<Error>
        {
            #region IEqualityComparer<Error> Members

            public bool Equals(Error x, Error y)
            {
                int c1, c2, c3, c4, c5, c6, c7, c8, c9, c10;
                try
                {
                    c1 = x.Active.CompareTo(y.Active);
                }
                catch { c1 = 0; }
                try
                {
                    c2 = x.Code.CompareTo(y.Code);
                }
                catch { c2 = 0; }
                try
                {
                    c3 = (x.D10Id == null && y.D10Id == null) ? 0 : x.D10Id.Value.CompareTo(y.D10Id.Value);
                }
                catch { c3 = 0; }
                try
                {
                    c4 = x.Description.CompareTo(y.Description);
                }
                catch { c4 = 0; }
                try
                {
                    c5 = x.Detail.CompareTo(y.Detail);
                }
                catch { c5 = 0; }
                try
                {
                    c6 = x.EdiSet.CompareTo(y.EdiSet);
                }
                catch { c6 = 0; }
                try
                {
                    c7 = x.ErrorId.CompareTo(y.ErrorId);
                }
                catch { c7 = 0; }
                try
                {
                    c8 = x.Message.CompareTo(y.Message);
                }
                catch { c8 = 0; }
                try
                {
                    c9 = x.System.CompareTo(y.System);
                }
                catch { c9 = 0; }
                try
                {
                    c10 = x.VIN.CompareTo(y.VIN);
                }
                catch { c10 = 0; }

                return (c1 + c2 + c3 + c4 + c5 + c6 + c7 + c8 + c9 + c10) == 0;
            }

            public int GetHashCode(Error obj)
            {
                return (obj.Active.ToString() + obj.Code.ToString() + obj.D10Id.ToString() + obj.Description.ToString() +
                    obj.Detail.ToString() + obj.EdiSet.ToString() + obj.Message.ToString() +
                    obj.System.ToString() + obj.VIN.ToString() + obj.X85Id.ToString()).ToLower().GetHashCode();
            }

            #endregion
        }

        public static void CheckErrorTableExists(ATLDbDataContext db, string connStr)
        {
            if (checkErrorTable)
            {
                string query = "SELECT COUNT(*) FROM SYSOBJECTS WHERE Name = 'Error'";
                using (SqlConnection conn = new SqlConnection(connStr))
                {
                    SqlCommand cmd = new SqlCommand(query, conn);
                    try
                    {
                        conn.Open();
                        int? result = (int?)cmd.ExecuteScalar();
                        if (result == null || result == 0)
                        {
                            //Create Error Table
                            SqlConnectionStringBuilder sb = new SqlConnectionStringBuilder(db.Connection.ConnectionString);
                            #region CREATE ERROR TABLE QUERY
                            query =
                                "USE [" + sb.InitialCatalog + "] " +
                                "GO " +
                                "CREATE TABLE [dbo].[Error]( " + 
                                "    [ErrorId] [int] IDENTITY(1,1) NOT NULL, " + 
                                "    [X85Id] [int] NULL, " +
                                "    [D10Id] [int] NULL, " +
                                "    [System] [varchar](30) NULL, " +
                                "    [EdiSet] [varchar](30) NULL, " +
                                "    [ErrorDateTime] [datetime] NULL, " +
                                "    [VIN] [varchar](17) NULL, " +
                                "    [Code] [varchar](25) NULL, " +
                                "    [Message] [varchar](100) NULL, " +
                                "    [Description] [varchar](2000) NULL, " +
                                "    [Detail] [varchar](2000) NULL, " +
                                "    [FilePath] [varchar](500) NULL, " +
                                "    [Active] [bit] NOT NULL, " +
                                "    CONSTRAINT [PK_Error] PRIMARY KEY CLUSTERED " +
                                "    ( " +
                                "        [ErrorId] ASC " +
                                "    ) ON [PRIMARY] " +
                                ") ON [PRIMARY] TEXTIMAGE_ON [PRIMARY] " +
                                "GO " +
                                "ALTER TABLE [dbo].[Error] ADD CONSTRAINT [DF_Error_Active] DEFAULT ((1)) FOR [Active] " +
                                "GO";
                            #endregion

                            string[] commands = query.Split(new string[] { "GO", "GO ", "GO\r\n", "GO\n", "GO\t" }, StringSplitOptions.RemoveEmptyEntries);
                            foreach (string c in commands)
                            {
                                cmd = new SqlCommand(c, conn);
                                cmd.ExecuteNonQuery();
                            }
                        }
                    }
                    catch (Exception ex)
                    {
                        AppLog.WriteExceptionToLog(ex, "Unable to verify Error table", true);
                    }
                }

                checkErrorTable = false;
            }
        }

        public static void AddErrorEntry(Error err)
        {
            errors.Add(err);
        }

        public static string GetErrorList()
        {
            string rtn = "";
            foreach (Error err in errors)
            {
                rtn += string.Format("{0}\t{1}\t{2}\t{3}\t{4}{5}{6}{7}{8}",
                    err.Message,
                    err.Description,
                    err.VIN,
                    err.Code,
                    err.EdiSet,
                    Environment.NewLine,
                    err.Detail,
                    Environment.NewLine,
                    err.FilePath);
            }
            return rtn;
        }

        public static void EmailErrorReport()
        {
            if (errors.Count < 1)
                return;

            SendEmail("noreply@atlinksystem.com", errorEmailAddr, "ACES Error Report", GetErrorList());
        }

        public static void SendEmail(string body)
        {
            SendEmail("noreply@atlinksystem.com", errorEmailAddr, "ACES Report", body);
        }

        public static void SendEmail(string subject, string body)
        {
            SendEmail("noreply@atlinksystem.com", errorEmailAddr, subject, body);
        }

        public static void SendEmail(string from, string to, string subject, string body)
        {
            try
            {
                MailMessage message = new MailMessage();
                SmtpClient client = new SmtpClient("smtp.gmail.com", 587);
                client.EnableSsl = true;
                client.Credentials = new NetworkCredential("braden@atlinksystem.com", "bartlet00");
                message.From = new MailAddress(from);
                message.To.Add(to);
                message.Subject = subject;
                message.Body = DateTime.Now.ToString() + Environment.NewLine + Environment.NewLine;
                message.Body += body;

                client.Send(message);
            }
            catch (Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, null, true);
            }
        }

        public static int GetNextSequenceNumber(ATLDbDataContext db)
        {
            int rtn = 0;
            int highDICN = 0;

            try
            {
                var x = (from p in db.X00s
                         where p.DataFormatType.ToUpper() == "ACES"
                         select p);

                if (x.Count() > 0)
                {
                    foreach (X00 x00 in x)
                    {
                        if (x00.DICN != null)
                        {
                            if (x00.DICN > highDICN)
                                highDICN = x00.DICN.Value;
                        }
                    }
                }
            }
            catch(Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, null, true);
            }

            try
            {
                X85 x85 = null;
                try
                {
                    x85 = (from p in db.X85s
                           where p.DataFormatType.ToUpper().Equals("ACES") &&
                                p.Processed == 1
                           orderby p.ProcessedTimeString descending
                           select p).FirstOrDefault() as X85;
                }
                catch (Exception ex) { AppLog.WriteExceptionToLog(ex, "No X85 found.", true); }

                if (x85 != null && !didResetDICN && x85.ProcessedTimeString != null &&
                         (DateTime.Now.Month != ((DateTime)x85.ProcessedTimeString).Month))
                {
                    highDICN = 0;
                    didResetDICN = true;
                }
                else if (x85 != null)
                    rtn = highDICN;

                var x = (from p in db.X00s
                         where p.DataFormatType.ToUpper() == "ACES"
                         select p);

                foreach (X00 x00 in x)
                {
                    x00.DICN = highDICN + 1; //todo: was set to highDICN++ which seems wrong..
                }

            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }
            finally
            {
                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
            }

            return rtn;
        }

        public static int CalculatePercentage(int count, int total)
        {
            int rtn = (int)(((float)count / total) * 100f);
            if (rtn > 100)
                rtn = 100;
            else if (rtn < 0)
                rtn = 0;
            return rtn;
        }

        public static int CalculatePercentage(long count, long total)
        {
            int rtn = (int)(((float)count / total) * 100f);
            if (rtn > 100)
                rtn = 100;
            else if (rtn < 0)
                rtn = 0;
            return rtn;
        }

        public static void MovePendingFile(FileInfo file)
        {
            try
            {
                if (!Directory.Exists("ACES"))
                    Directory.CreateDirectory("ACES");
                if (!Directory.Exists("ACES\\Pending"))
                    Directory.CreateDirectory("ACES\\Pending");

                file.CopyTo(string.Format("{0}{1}",
                    "ACES\\Pending\\", file.Name),
                    true);
            }
            catch (Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, null, true);
            }
        }

        public static void DecodeAndPriceD10(D10 d10, ATLDbDataContext db, ref decimal runningTotalPrice)
        {
            try
            {
                string year = string.Empty, make = string.Empty,
                    model = string.Empty, options = string.Empty;
                db.sp_decode_vin(d10.VIN, ref year, ref make, ref model, ref options);
                d10.Year = year;
                d10.Make = make;
                d10.Model = model;
                d10.Options = options;

                db.SubmitChanges();
            }
            catch (Exception ex)
            {
                AppLog.WriteToLog(string.Format("Unable to decode VIN: {0}", d10.VIN));
            }

            try
            {
                decimal? returnRate = null, fuelSurcharge = null;
                string fuelCalcType = string.Empty;
                db.sp_price_vin(d10.D10Id, ref returnRate, ref fuelSurcharge, ref fuelCalcType);

                d10.TransportAmount = returnRate;
                d10.FuelSurcharge = fuelSurcharge;
                d10.FuelCalcType = fuelCalcType;

                runningTotalPrice += (decimal)returnRate;
                decimal fuel = 0M;
                if (fuelCalcType.Equals("PER UNIT", StringComparison.InvariantCultureIgnoreCase))
                {
                    runningTotalPrice += (decimal)fuelSurcharge;
                }
                else
                {
                    fuel = (((decimal)fuelSurcharge) * 0.01M) * ((decimal)returnRate);
                    runningTotalPrice += fuel;
                }
            }
            catch (Exception ex)
            {
                AppLog.WriteToLog(string.Format("Unable to price VIN: {0}", d10.VIN));
            }

            db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
        }

        internal static void RestoreArchives(bool overwrite)
        {
            try
            {
                if (Directory.Exists("ACES"))
                {
                    if (Directory.Exists("ACES\\Download_Backup"))
                    {
                        if (!Directory.Exists("ACES\\Restore\\Download"))
                            Directory.CreateDirectory("ACES\\Restore\\Download");
                        string[] dFiles = Directory.GetFiles("ACES\\Download_Backup\\");
                        foreach (string file in dFiles)
                        {
                            FileInfo f = new FileInfo(file);
                            int startIndex = f.Name.IndexOf("G", StringComparison.InvariantCultureIgnoreCase);
                            int endIndex = f.Name.LastIndexOf(".");
                            string newName = f.Name.Substring(startIndex, endIndex - startIndex);
                            string defaultName = f.Name.Substring(startIndex, endIndex - startIndex);
                            int fileCount = 2;
                            while (!overwrite && File.Exists("ACES\\Restore\\Download\\" + newName + ".txt"))
                            {
                                newName = string.Format("{0}({1})", defaultName, fileCount++);
                            }

                            try
                            {
                                f.CopyTo(string.Format("{0}{1}",
                                    "ACES\\Restore\\Download\\",
                                    newName + ".txt"),
                                    overwrite);
                            }
                            catch (Exception ex)
                            {
                                AppLog.WriteExceptionToLog(ex, "Unable to copy files", false);
                            }
                        }
                    }

                    if (Directory.Exists("ACES\\Upload_Backup"))
                    {
                        if (!Directory.Exists("ACES\\Restore\\Upload"))
                            Directory.CreateDirectory("ACES\\Restore\\Upload");
                        string[] uFiles = Directory.GetFiles("ACES\\Upload_Backup\\");
                        foreach (string file in uFiles)
                        {
                            FileInfo f = new FileInfo(file);

                            int startIndex = f.Name.IndexOf("R", StringComparison.InvariantCultureIgnoreCase);
                            int endIndex = f.Name.LastIndexOf(".");
                            string newName = f.Name.Substring(startIndex, endIndex - startIndex);
                            string defaultName = f.Name.Substring(startIndex, endIndex - startIndex);
                            int fileCount = 2;
                            while (!overwrite && File.Exists(newName + ".txt"))
                            {
                                newName = string.Format("{0}({1})", defaultName, fileCount++);
                            }

                            try
                            {
                                f.CopyTo(string.Format("{0}{1}",
                                    "ACES\\Restore\\Download\\",
                                    newName + ".txt"),
                                    overwrite);
                            }
                            catch (Exception ex)
                            {
                                AppLog.WriteExceptionToLog(ex, "Unable to copy files", false);

                            }
                        }
                    }
                }
            }
            catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }
        }

        public class StringPathComparer : IComparer<string>
        {
            public bool hasDate = true;
            #region IComparer<string> Members

            public int Compare(string x, string y)
            {
                int rtn = 0;

                if (hasDate)
                {
                    string xDatePart = x.Substring(15, 14);
                    string yDatePart = y.Substring(15, 14);
                    DateTime xOperationDateTime, yOperationDateTime;
                    if (DateTime.TryParseExact(xDatePart, "yyyyMMddHHmmss", CultureInfo.InvariantCulture,
                        DateTimeStyles.None, out xOperationDateTime) && DateTime.TryParseExact(yDatePart,
                        "yyyyMMddHHmmss", CultureInfo.InvariantCulture, DateTimeStyles.None, out yOperationDateTime))
                    {
                        rtn = xOperationDateTime.CompareTo(yOperationDateTime);
                    }
                }
                
                if(rtn == 0)
                {
                    int sidx = hasDate ? 30 : 15;
                    FileTypeValue type1 = (FileTypeValue)Enum.Parse(typeof(FileTypeValue), x.Substring(sidx, 3));
                    FileTypeValue type2 = (FileTypeValue)Enum.Parse(typeof(FileTypeValue), y.Substring(sidx, 3));

                    int calc = type1 - type2;
                    if (calc < 0)
                        rtn = -1;
                    else if (calc == 0)
                    {
                        rtn = x.CompareTo(y);
                    }
                    else
                        rtn = 1;  
                }

                return rtn;
            }

            #endregion

            internal enum FileTypeValue
            {
                G80 = 0,
                G08 = 1,
                G73 = 2,
                G78 = 3,
                G70 = 4,
                G95 = 5,
                G07 = 6,
                G92 = 7,
                G96 = 8,
                G50 = 9,
                G51 = 10,
            }
        }


        public static void Reprocess(ATLDbDataContext db, string connStr)
        {
            Console.WriteLine("");
            DateTime start = DateTime.Now;
            if (!Directory.Exists("ACES\\Reprocess\\"))
            {
                Console.WriteLine("Creating Reprocess directory");
                Directory.CreateDirectory("ACES\\Reprocess\\");
                return;
            }

            bool secondary = false;
            Console.WriteLine("Collecting file information");
            string[] files = Directory.GetFiles("ACES\\Reprocess\\", "???????????????G*.txt", SearchOption.TopDirectoryOnly);
            if (files.Length == 0)
            {
                secondary = true;
                files = Directory.GetFiles("ACES\\Reprocess\\", "G*.txt", SearchOption.TopDirectoryOnly);
            }
            if (files.Length == 0)
            {
                Console.WriteLine("No files to process");
                AppLog.WriteToLog("No files to reprocess");
                return;
            }

            StringPathComparer spComparer = new StringPathComparer();
            spComparer.hasDate = !secondary;
            Array.Sort<string>(files, spComparer);

            int fileIndexStart = secondary ? 0 : 15;
            foreach (string file in files)
            {
                Console.WriteLine("Reprocessing {0}", file);
                string namePart = file.Substring(fileIndexStart + 15);
                string datePart = string.Empty;
                DateTime operationDateTime;
                if (!secondary)
                {
                    datePart = file.Substring(fileIndexStart + 0, 14);
                    if (!DateTime.TryParseExact(datePart, "yyyyMMddHHmmss", CultureInfo.InvariantCulture,
                        DateTimeStyles.None, out operationDateTime))
                        operationDateTime = start;
                }
                else
                    operationDateTime = start;

                TransmissionInfo info = new TransmissionInfo();
                info.FileName = namePart;
                info.LocalFile = new FileInfo(file);

                try
                {
                    switch (namePart.Substring(0,3))
                    {
                        case "G07":
                            G07 g07 = G07.Load(info, false);
                            g07.CreatedDateTime = operationDateTime;
                            g07.Process(db);
                            break;
                        case "G08":
                            G08 g08 = G08.Load(info, false);
                            g08.CreatedDateTime = operationDateTime;
                            g08.Process(db);
                            break;
                        case "G50":
                            G50 g50 = G50.Load(info, false);
                            g50.CreatedDateTime = operationDateTime;
                            g50.Process(db);
                            foreach (G50Detail d in g50.Detail)
                            {
                                Error err = new Error();
                                err.Message = "ACES Rejection";
                                err.Description = d.RejectExplanation.Value;
                                err.Code = d.RejectCode.Value;
                                err.EdiSet = d.FileNameToResend.Value.Substring(0, 3);
                                err.System = "ACES";
                                err.ErrorDateTime = operationDateTime;
                                err.FilePath = d.FileNameToResend.Value + "." + d.FileNameExtension.Value;
                                err.Detail = d.ToString();
                                err.Active = true;

                                Utils.AddErrorEntry(err);
                            }
                            break;
                        case "G51":
                            G51 g51 = G51.Load(info, false);
                            g51.CreatedDateTime = operationDateTime;
                            g51.Process(db);
                            foreach (G51Detail d in g51.Detail)
                            {
                                Error err = new Error();
                                err.Message = "ACES Rejection";
                                err.Description = d.ErrorCodeDescription.Value;
                                err.Code = d.ErrorCode.Value;
                                err.EdiSet = d.FileName.Value.Substring(0, 3);
                                err.System = "ACES";
                                err.ErrorDateTime = operationDateTime;
                                err.FilePath = d.FileName.Value;
                                err.VIN = d.VIN.Value;
                                err.Detail = d.ToString();
                                err.Active = true;

                                Utils.AddErrorEntry(err);
                            }
                            break;
                        case "G70":
                            G70 g70 = G70.Load(info, false);
                            g70.CreatedDateTime = operationDateTime;
                            g70.Process(db);
                            break;
                        case "G73":
                            G73 g73 = G73.Load(info, false);
                            g73.CreatedDateTime = operationDateTime;
                            g73.Process(db, true);
                            break;
                        case "G78":
                            G78 g78 = G78.Load(info, false);
                            g78.CreatedDateTime = operationDateTime;
                            g78.Process(db);
                            break;
                        case "G92":
                            G92 g92 = G92.Load(info, false);
                            g92.CreatedDateTime = operationDateTime;
                            g92.Process(db);
                            break;
                        case "G95":
                            G95 g95 = G95.Load(info, false);
                            g95.CreatedDateTime = operationDateTime;
                            g95.Process(db);
                            break;
                        case "G96":
                            G96 g96 = G96.Load(info, false);
                            g96.CreatedDateTime = operationDateTime;
                            g96.Process(db);
                            break;
                        default:
                            break;
                    }

                    info.LocalFile.Delete();
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }
            }

            Utils.LogAllErrors(db, connStr);
        }
    }
}
