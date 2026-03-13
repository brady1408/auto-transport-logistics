
<%@ Language=VBScript %>
<%option explicit%>

<%

' ****
' APPLICATION SPECIFIC CONSTANTS
' ****
CONST ERR_SRC				= "W2kSpokeReceive.asp"
CONST ERR_CONTENT_TYPE			= 10001
CONST ERR_POSTED_DOCUMENT		= 10002
CONST ERR_SEND_TO_QUEUE			= 10003
CONST ERR_CREATE_ESPOKE_OBJECT	= 10004
CONST ERR_GET_PAYLOAD			= 10005
CONST ERR_RECEIPT			= 10006

' ****
' ADODB CONSTANTS
' ****
CONST ERR_ADODB_STREAM_CREATE		= 10007

' ****
' XMLDOM CONSTANTS
' ****
CONST ERR_CREATE_XMLDOM_OBJECT		= 10008

' ****
' HTTP CONSTANTS
' ****

CONST HTTP_STATUS_NOT_ACCEPTABLE	= "404 Not Found"
CONST HTTP_STATUS_INTERNAL_ERROR	= "500 Internal server error"
CONST HTTP_STATUS_ACCEPTED			= "202 Accepted"

'***********************************************************************
'                     ITRAP EVENT LOGGING
'***********************************************************************
CONST ITRAP_EVENT_LOG_SRC		= "W2kSpokeReceive.asp"
CONST ITRAP_EVENT_CATEGORY		= 0
CONST ITRAP_EVENT_ERROR_ID		= &HC00003EA ' 1002

CONST ITRAP_EVENT_TYPE_ERROR		= 0
CONST ITRAP_EVENT_TYPE_WARNING		= 1
CONST ITRAP_EVENT_TYPE_INFORMATION	= 2


Function GetCSVFileName()
    Dim strHr, strMin, strSec, strRandom   'current time, used to generate a CSV filename
    Dim strCSVName
    
    strHr = Right("00" & Hour(Time()), 2)
    strMin = Right("00" & Minute(Time()), 2)
    strSec = Right("00" & Second(Time()), 2)
    strRandom = Right("0000" & Rnd, 4)
    strCSVName = "MSMQ_Entry" & "_" & strHr & strMin & strSec & strRandom & ".txt"

    GetCSVFileName = strCSVName
    
End Function

'________________________________________________________________________________________________________________
'	Method Name:	SendToDatabase
'	Description:	This method calls a COM wrapper comTSCBusinessObject which in turn calls		
'				a COM compliant assembly TechCenter.CDC.TSCBusinessObject to save TSCDATA sent
'				by QLS-CM.
'
'					The TechCenter.CDC.TSCBusinessObject assembly is the same assembly used by the 
'				2D scans, and the FTP process.
'				
'					The strQueueLabel effects how the SendToDatabase Method functions.  If the strQueueLabel = "input"
'				then the function \ page will operate without using eSpoke.  Else if the strQueueLabel = "payload" then
'				the function \ page will use eSpoke.
'________________________________________________________________________________________________________________
Sub SendToDatabase(strPostedDocument, strQueueLabel)
On Error Resume Next
    
    'Declare Variables
    Dim objTSCBusinessObject
    
    'Set Object
    Set objTSCBusinessObject = Server.CreateObject("comTSCBusinessObject.Process")
           
    If objTSCBusinessObject Is Nothing Then
        SendToQueue "Unable to create comTSCBusinessObject.Process object", "Send To Database"
    End If
    
    objTSCBusinessObject.SendToDatabase strPostedDocument, strQueueLabel

    If err.number <> 0 Then
		sMsg = "TSC Business Rules Failed  - Error Number: " & Err.number & " Description: " & Err.description
		LogError sMsg, ERR_RECEIPT, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	
        On Error Goto 0
		  Err.Raise ERR_RECEIPT, ERR_SRC, sMsg
	  Exit Sub
    End If
    
    'Destroy Objects
    Set objTSCBusinessObject = nothing
        
End Sub

'________________________________________________________________________________________________________________
'	Method Name:	SendToQueue
'	Description:	A development method to log eHub data to a unique flat file located at
'				c:\inetpub\ftproot.  This method used to store send data or error messages.	
'
'					Original method was designed by eHub to store data into MSMQ.
'
'________________________________________________________________________________________________________________

Sub SendToQueue(strPostedDocument, strQueueLabel)
	ON ERROR RESUME NEXT
  
	Dim objFSO,strFileName, f

	' Note: in production environment the code below must check for return values of MSMQ
	'	calls and if anything failed, return an error message back to the sending BizTalk Server
	'	using Response.Write().  This will assure that the sending server re-tries the transmission
	'	and the docuement is not lost 


	Set objFSO = Server.CreateObject("Scripting.FileSystemObject")
	strFileName = GetCSVFileName
	Set f = objFSO.CreateTextFile("E:\LogFiles\eHub\" & strFileName, true, false)

	f.Writeline Now
	f.Writeline strPostedDocument
	f.Writeline strQueueLabel

	f.close
	Set f=nothing 
	Set objFSO = nothing
	Set strFileName = nothing

	If err.number <> 0 Then
		sMsg = "Call to oQueueMsg.Send failed - Number: " & Err.number & _
    		   " Description: " & Err.description
    	LogError sMsg, ERR_MSMQ_MESSAGE_SEND_FAIL, ERR_SRC
    	On Error Goto 0
    	Err.Raise ERR_MSMQ_MESSAGE_SEND_FAIL, ERR_SRC, sMsg
    	Exit Sub
	End If

End Sub

'________________________________________________________________________________________________________________
'	Method Name:	LogToDatabase
'	Description:	A QA tool to trace eHub Transactions
'
'
'
'________________________________________________________________________________________________________________

Sub LogToDatabase(ErrorString, ErrorNumber, ErrorSource)
On Error Resume Next
    
    'Declare Variables
    Dim objTSCBusinessObject
    
    'Set Object
    Set objTSCBusinessObject = Server.CreateObject("comTSCBusinessObject.Process")
           
    If objTSCBusinessObject Is Nothing Then
        SendToQueue "Unable to create comTSCBusinessObject.Process object", "Log To Database"
    End If
    
    objTSCBusinessObject.LogToDatabase ErrorString, ErrorNumber, ErrorSource

    If err.number <> 0 Then
	SendToQueue err.number & " " & err.description, "Log To Database"
	sMsg = "Log To Database - Error Number: " & Err.number & " Description: " & Err.description
	LogError sMsg, ERR_RECEIPT, ERR_SRC
	Response.Status = HTTP_STATUS_INTERNAL_ERROR
	
        On Error Goto 0
	  Err.Raise ERR_RECEIPT, ERR_SRC, sMsg
	  Exit Sub
    End If
    
    'Destroy Objects
    set objTSCBusinessObject = nothing
        
End Sub



'________________________________________________________________________________________________________________
'	Method Name:	LogError
'	Description:	To trace errors to a flat file.  The files are saved to c:\inetpub\ftproot.
'
'
'
'________________________________________________________________________________________________________________
'
Sub LogError(ErrorString, ErrorNumber, ErrorSource)

	ON ERROR RESUME NEXT

	Dim sErrMsg
	Dim objFSO,strFileName, f

	sErrMsg = NOW & ", SOURCE: " & ErrorSource & ", ERROR #: " & ErrorNumber & ", DESCRIPTION: " & ErrorString 
	sErrMsg = ITRAP_EVENT_LOG_SRC & ", " & ITRAP_EVENT_TYPE_ERROR & ", " & ITRAP_EVENT_CATEGORY & ", " & ITRAP_EVENT_ERROR_ID & ", " & sErrMsg

	'Response.Write(sErrMsg & "<br>")
	Set objFSO = Server.CreateObject("Scripting.FileSystemObject")
	Set f = objFSO.OpenTextFile("c:\inetpub\ftproot\W2KeSpoke.log", 8, true)

	If err.number <> 0 Then
		Response.Write("Error writing log file")
    	Exit sub
	End If

	f.Writeline sErrMsg

	f.close
	Set f=nothing 
	Set objFSO = nothing
	Set strFileName = nothing

End Sub
'_________________________________________________________________________________________________________________

Function GetContentType()
On Error Resume Next
	Dim strCharSet,strContentType
	Dim nStartPos,nEndPos,strContentType1
	
	strContentType = Request.ServerVariables( "CONTENT_TYPE" )
  	'
	' Determine request entity body character set (default to us-ascii)
	'
	strCharSet = "us-ascii"
	nStartPos = InStr( 1, strContentType, "CharSet=""", 1)
	If (nStartPos <> 0 ) then
		nStartPos = nStartPos + Len("CharSet=""")
		nEndPos = InStr( nStartPos, strContentType, """",1 )
		strCharSet = Mid (strContentType, nStartPos, nEndPos - nStartPos )
	End if

	if ( strContentType = "" or Request.TotalBytes = 0) then
		'
		' Content-Type is required as well as an entity body
		'
		sMsg = "Content-type or Entity body is missing" & VbCrlf & _
			   "Message headers follow below:"  & VbCrlf & _
			   Request.ServerVariables("ALL_RAW")
		LogError sMsg, ERR_CONTENT_TYPE, ERR_SRC
		Response.Status = HTTP_STATUS_NOT_ACCEPTABLE
		On Error Goto 0
		Err.Raise ERR_CONTENT_TYPE, ERR_SRC, sMsg
		Exit Function
	End If

    GetContentType = strCharSet
end function
'_________________________________________________________________________________________________________________
Function GetPostedDocument(strCharSet)
	On Error Resume Next

	Dim strPostedDocument
	Dim vtEntityBody
	Dim oStream
	Dim sMsg
	Dim strContentType
	
	strPostedDocument = ""

	vtEntityBody = Request.BinaryRead (Request.TotalBytes )
	
 	'
	' Convert to UNICODE
	'
	Set oStream = Server.CreateObject("AdoDB.Stream")
	If err.number <> 0 Then
		sMsg = "Create object for ADODB.Stream failed - Number: " & Err.number & _
    		   " Description: " & Err.description
    	LogError sMsg, ERR_ADODB_STREAM_CREATE, ERR_SRC
    	On Error Goto 0
    	Err.Raise ERR_ADODB_STREAM_CREATE, ERR_SRC, sMsg
    	Exit Function
	End If
	oStream.Type = 1						'adTypeBinary
	oStream.Open
	oStream.Write vtEntityBody
	oStream.Position = 0
	oStream.Type = 2						'adTypeText
	oStream.Charset = strCharSet
	strPostedDocument = strPostedDocument & oStream.ReadText
	oStream.Close
	Set oStream = Nothing
	GetPostedDocument = strPostedDocument
	
End Function
'________________________________________________________________________________________________________
Function LoadPostedDocument()
	On Error Resume Next
	Dim objXML

	Set objXML = Server.CreateObject("Microsoft.XMLDOM")	
	
	objXML.load(Request)

	If err.number <> 0 Then
		sMsg = "Load into DOM failed - Number: " & Err.number & _
    		   " Description: " & Err.description
    		LogError sMsg, ERR_CREATE_XMLDOM_OBJECT, ERR_SRC
	    	On Error Goto 0
    		Err.Raise ERR_CREATE_XMLDOM_OBJECT, ERR_SRC, sMsg
	    	Exit Function
	End If

	LoadPostedDocument = objXML.xml

End Function
'____________________________________________________________________________________________________________
Sub Main()
	On Error Resume Next
	
	Dim strCharset
	Dim sPostedDocument
	Dim sMsg
	Dim strReceipt
	Dim oESpokeObj
	Dim strPayload

	' REMOVE IN PRODUCTION - Write IP 
	LogToDatabase "QLS-CM was here " & Request.ServerVariables("REMOTE_HOST"), 0, ERR_SRC

	strCharset = GetContentType()
	If err.number <> 0 Then
		sMsg = "Could not get Content type : " & Err.number & _
    		   " Description: " & Err.description
		LogToDatabase sMsg, ERR_CONTENT_TYPE, ERR_SRC
		Response.Status = HTTP_STATUS_NOT_ACCEPTABLE
		On Error Goto 0
		Err.Raise ERR_CONTENT_TYPE, ERR_SRC, sMsg
	End If

	' REMOVE IN PRODUCTION
	'SendToQueue strCharset, "ContentType"
	sPostedDocument = GetPostedDocument(strCharset)
	If err.number <> 0 Then
		sMsg = "Could not get the posted document with charset : " & strCharset & " Error Number -" &Err.number & _
    		   " Description: " & Err.description
		LogToDatabase sMsg, ERR_POSTED_DOCUMENT, ERR_SRC
		Response.Status = HTTP_STATUS_NOT_ACCEPTABLE
		On Error Goto 0
		Err.Raise ERR_POSTED_DOCUMENT, ERR_SRC, sMsg
	End If

	' REMOVE IN PRODUCTION
	'SendToQueue sPostedDocument, "Input"
	
	' If eSpoke has a failure pass in input document and pull data out.
	'SendToDatabase sPostedDocument, "Input"

	If err.number <> 0 Then
		sMsg = "Could not post Input document to queue - Error Number: " & Err.number & " Description: " & Err.description
		LogToDatabase sMsg, ERR_SEND_TO_QUEUE, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_SEND_TO_QUEUE, ERR_SRC, sMsg
	    Exit Sub
	End If

	set oESpokeObj = Server.CreateObject("ESpoke.Receiver")
	If err.number <> 0 Then
		sMsg = "ESpoke.Receiver object not created  - Error Number: " & Err.number & " Description: " & Err.description
		LogToDatabase sMsg, ERR_CREATE_ESPOKE_OBJECT, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_CREATE_ESPOKE_OBJECT, ERR_SRC, sMsg
	    Exit Sub
	End If

    oESpokeObj.Charset = "us-ascii"
    oESpokeObj.PblCert = "6becd2176b301868c29953aad439268e31e0d8a7" ' ehub 477.xml contract
    oESpokeObj.PblStore = "My"
    oESpokeObj.PvtCert = "" 'signed and encrypted message retrieval and signed receipts
    oESpokeObj.PvtStore = "My"

	strPayload = oESpokeObj.Payload(sPostedDocument)
	If err.number <> 0 Then
		sMsg = "Couldnot get payload from eSpoke - Error Number: " & Err.number & " Description: " & Err.description
		LogToDatabase sMsg, ERR_GET_PAYLOAD, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_GET_PAYLOAD, ERR_SRC, sMsg
	    Exit Sub
	End If

	' REMOVE IN PRODUCTION
	SendToQueue strPayload, "Payload"
	
	'Send XML Document to CDC II (Use Payload instead of Input)
	'SendToDatabase sPostedDocument, "Payload"

	'Send to CDC II
	SendToDatabase sPostedDocument, "Input"
	
	If err.number <> 0 Then
		sMsg = "Could not post Payload document to queue - Error Number: " & Err.number & " Description: " & Err.description
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_SEND_TO_QUEUE, ERR_SRC, sMsg
	    Exit Sub
	End If

	strReceipt = oESpokeObj.Receipt
	If err.number <> 0 Then
		sMsg = "Receipt Retrieval Failed  - Error Number: " & Err.number & " Description: " & Err.description
		LogToDatabase sMsg, ERR_RECEIPT, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_RECEIPT, ERR_SRC, sMsg
	    Exit Sub
	End If

	' REMOVE IN PRODUCTION
	'SendToQueue strReceipt, "Receipt"
	if strReceipt <> "" then 
		Response.write strReceipt	
		If err.number <> 0 Then
			sMsg = "Could not post Payload document to queue - Error Number: " & Err.number & " Description: " & Err.description
			Response.Status = HTTP_STATUS_INTERNAL_ERROR
			On Error Goto 0
			Err.Raise ERR_SEND_TO_QUEUE, ERR_SRC, sMsg
			Exit Sub
		End If
	end if

	set oESpokeObj = Nothing
	Response.Status = HTTP_STATUS_ACCEPTED
	Response.end
	
End Sub
'____________________________________________________________________________________________________________________________________________________________

	' call the main sub
	On Error Resume Next
'	Response.Status = HTTP_STATUS_ACCEPTED
'	Response.end

	call Main()
%>