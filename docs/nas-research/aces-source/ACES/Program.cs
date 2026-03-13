using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows.Forms;
using System.Data.SqlClient;
using System.Net;
using System.IO;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    static class Program
    {
        /// <summary>
        /// The main entry point for the application.
        /// </summary>
        [STAThread]
        static void Main(string[] args)
        {
            if (args.Length > 0 && args[0].Equals("/r"))
            {
                bool overwrite = false;
                if (args.Length > 1)
                {
                    if (args[1].Equals("-o"))
                        overwrite = true;
                }
                Utils.RestoreArchives(overwrite);
            }

            if (args.Length > 0 && args[0].Equals("/p"))
            {
                if (args.Length > 4)
                {
                    SqlConnectionStringBuilder sb = new SqlConnectionStringBuilder();
                    sb.DataSource = args[1];
                    sb.InitialCatalog = args[2];
                    sb.UserID = args[3];
                    sb.Password = args[4];

                    ATLDbDataContext db = new ATLDbDataContext(sb.ConnectionString);
                    #if(DEBUG)
                        TextWriter log;
                        log = File.AppendText("reprocessdebuglog.sql");
                        db.Log = log;
                    #endif
                    Utils.Reprocess(db, sb.ConnectionString);
                    return;
                }
            }

            if (args.Length < 3)
                return;
            
            SqlConnectionStringBuilder sqlsb = new SqlConnectionStringBuilder();
            sqlsb.DataSource = args[0];
            sqlsb.InitialCatalog = args[1];

            bool conversion = false, exitOnComplete = false, debugMode = false, useWindowsAuth = false;
            DateTime convStart = DateTime.MinValue, convEnd = DateTime.MaxValue;
            for (int i = 0; i < args.Length; i++)
            {
                switch (args[i].ToLowerInvariant())
                {
                    /* Usage:
                     * /c [2007-03-05|2009-04-01] /e /d
                     */
                    case "/c":
                        conversion = true;
                        if ((i + 1) < args.Length && args[i + 1].StartsWith("[") && args[i + 1].EndsWith("]"))
                        {
                            i++;
                            string timespan = args[i].Trim(new char[] { '[', ']', ' ' });
                            string[] dates = timespan.Split(new char[] { '|' }, 2, StringSplitOptions.RemoveEmptyEntries);
                            if (dates.Length == 2)
                            {
                                DateTime.TryParseExact(dates[0], "yyyy-MM-dd", CultureInfo.InvariantCulture,
                                    DateTimeStyles.NoCurrentDateDefault, out convStart);
                                DateTime.TryParseExact(dates[1], "yyyy-MM-dd", CultureInfo.InvariantCulture,
                                    DateTimeStyles.NoCurrentDateDefault, out convEnd);
                            }
                        }
                        continue;
                    case "/e":
                        exitOnComplete = true;
                        continue;
                    case "/d":
                        debugMode = true;
                        continue;
                    case "/w":
                        useWindowsAuth = true;
                        continue;
                    default:
                        continue;
                }
            }

            if (!useWindowsAuth)
            {
                sqlsb.UserID = args[2];
                sqlsb.Password = args[3];
            }
            else
                sqlsb.IntegratedSecurity = true;
            

            Application.EnableVisualStyles();
            Application.SetCompatibleTextRenderingDefault(false);
            if (convStart != DateTime.MinValue && convEnd != DateTime.MaxValue)
                Application.Run(new ACESMainForm(sqlsb.ConnectionString, conversion, convStart, convEnd,
                    exitOnComplete, debugMode));
            Application.Run(new ACESMainForm(sqlsb.ConnectionString, conversion, exitOnComplete, debugMode));
        }
    }
}
