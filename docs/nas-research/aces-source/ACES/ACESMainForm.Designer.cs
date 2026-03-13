namespace ATLNetwork.EDI.ACES
{
    partial class ACESMainForm
    {
        /// <summary>
        /// Required designer variable.
        /// </summary>
        private System.ComponentModel.IContainer components = null;

        /// <summary>
        /// Clean up any resources being used.
        /// </summary>
        /// <param name="disposing">true if managed resources should be disposed; otherwise, false.</param>
        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null))
            {
                components.Dispose();
            }
            base.Dispose(disposing);
        }

        #region Windows Form Designer generated code

        /// <summary>
        /// Required method for Designer support - do not modify
        /// the contents of this method with the code editor.
        /// </summary>
        private void InitializeComponent()
        {
            this.label_status = new System.Windows.Forms.Label();
            this.progressBar_main = new System.Windows.Forms.ProgressBar();
            this.label_progress = new System.Windows.Forms.Label();
            this.panel_labels = new System.Windows.Forms.Panel();
            this.button_close = new System.Windows.Forms.Button();
            this.panel_labels.SuspendLayout();
            this.SuspendLayout();
            // 
            // label_status
            // 
            this.label_status.AutoSize = true;
            this.label_status.Dock = System.Windows.Forms.DockStyle.Left;
            this.label_status.Location = new System.Drawing.Point(1, 3);
            this.label_status.Name = "label_status";
            this.label_status.Size = new System.Drawing.Size(61, 13);
            this.label_status.TabIndex = 0;
            this.label_status.Text = "Initializing...";
            // 
            // progressBar_main
            // 
            this.progressBar_main.Location = new System.Drawing.Point(13, 30);
            this.progressBar_main.Name = "progressBar_main";
            this.progressBar_main.Size = new System.Drawing.Size(559, 23);
            this.progressBar_main.Style = System.Windows.Forms.ProgressBarStyle.Continuous;
            this.progressBar_main.TabIndex = 1;
            // 
            // label_progress
            // 
            this.label_progress.AutoSize = true;
            this.label_progress.Dock = System.Windows.Forms.DockStyle.Right;
            this.label_progress.Location = new System.Drawing.Point(543, 3);
            this.label_progress.Name = "label_progress";
            this.label_progress.Size = new System.Drawing.Size(16, 13);
            this.label_progress.TabIndex = 2;
            this.label_progress.Text = "...";
            // 
            // panel_labels
            // 
            this.panel_labels.Controls.Add(this.label_progress);
            this.panel_labels.Controls.Add(this.label_status);
            this.panel_labels.Location = new System.Drawing.Point(12, 12);
            this.panel_labels.Name = "panel_labels";
            this.panel_labels.Padding = new System.Windows.Forms.Padding(1, 3, 1, 3);
            this.panel_labels.Size = new System.Drawing.Size(560, 20);
            this.panel_labels.TabIndex = 3;
            // 
            // button_close
            // 
            this.button_close.Enabled = false;
            this.button_close.Location = new System.Drawing.Point(255, 78);
            this.button_close.Name = "button_close";
            this.button_close.Size = new System.Drawing.Size(75, 23);
            this.button_close.TabIndex = 4;
            this.button_close.Text = "Close";
            this.button_close.UseVisualStyleBackColor = true;
            this.button_close.Click += new System.EventHandler(this.button_close_Click);
            // 
            // ACESMainForm
            // 
            this.AutoScaleDimensions = new System.Drawing.SizeF(6F, 13F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(584, 113);
            this.Controls.Add(this.button_close);
            this.Controls.Add(this.panel_labels);
            this.Controls.Add(this.progressBar_main);
            this.FormBorderStyle = System.Windows.Forms.FormBorderStyle.FixedDialog;
            this.MaximizeBox = false;
            this.MinimizeBox = false;
            this.Name = "ACESMainForm";
            this.ShowIcon = false;
            this.StartPosition = System.Windows.Forms.FormStartPosition.CenterScreen;
            this.Text = "ACES";
            this.Load += new System.EventHandler(this.ACESMainForm_Load);
            this.panel_labels.ResumeLayout(false);
            this.panel_labels.PerformLayout();
            this.ResumeLayout(false);

        }

        #endregion

        private System.Windows.Forms.Label label_status;
        private System.Windows.Forms.ProgressBar progressBar_main;
        private System.Windows.Forms.Label label_progress;
        private System.Windows.Forms.Panel panel_labels;
        private System.Windows.Forms.Button button_close;
    }
}

