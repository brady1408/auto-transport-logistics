
<%@ Language=VBScript %>
<%option explicit%>

<%
' ****
' XMLDOM CONSTANTS
' ****
CONST ERR_CREATE_XMLDOM_OBJECT		= 10008


'***********************************************************************
'                     MSMQ CONSTANTS
'***********************************************************************

Const MQ_SEND_ACCESS				= 2
Const MQ_DENY_NONE					= 0

CONST ERR_MSMQ_QUEUEINFO_CREATE		= 11001
CONST ERR_QUEUEINFO_FORMATNAME_FAIL = 11002
CONST ERR_QUEUEINFO_OPEN_FAIL		= 11003
CONST ERR_MSMQ_MESSAGE_CREATE		= 11004
CONST ERR_MSMQ_MESSAGE_BODY_FAIL	= 11005
CONST ERR_MSMQ_MESSAGE_SEND_FAIL	= 11006


' ****
' HTTP CONSTANTS
' ****

CONST HTTP_STATUS_NOT_ACCEPTABLE	= "406 Not Acceptable"
CONST HTTP_STATUS_INTERNAL_ERROR	= "500 Internal server error"
CONST HTTP_STATUS_ACCEPTED			= "202 Accepted"

'***********************************************************************
'                     ITRAP EVENT LOGGING
'***********************************************************************
CONST ITRAP_EVENT_LOG_SRC			= "W2kSpokeReceive.asp"
CONST ITRAP_EVENT_CATEGORY			= 0
CONST ITRAP_EVENT_ERROR_ID			= &HC00003EA ' 1002

CONST ITRAP_EVENT_TYPE_ERROR		= 0
CONST ITRAP_EVENT_TYPE_WARNING		= 1
CONST ITRAP_EVENT_TYPE_INFORMATION	= 2

'***********************************************************************
'                     ESPOKE CONSTANTS
'***********************************************************************
CONST ERR_SRC			= "PublisheSpoke.asp"
CONST ERR_CONTENT_TYPE			= 0
CONST ERR_ADODB_STREAM_CREATE			= 1' 1002

CONST ERR_SEND_TO_QUEUE		= 2
CONST ERR_CREATE_ESPOKE_OBJECT		= 3
CONST ERR_GET_PAYLOAD	= 4
CONST ERR_RECEIPT	= 5
CONST ERR_STATUS	= 6

'_________________________________________________________________________________________________________________
'________________________________________________________________________________________________________________
'
Sub LogError(ErrorString, ErrorNumber, ErrorSource)

	ON ERROR RESUME NEXT

	Dim oEventLog
	Dim sErrMsg

	sErrMsg = "SOURCE: " & ErrorSource & ", ERROR #: " & ErrorNumber & ", DESCRIPTION: " & ErrorString

	' create the event log object
	Set oEventLog = Server.CreateObject("ITRAPFX.EventLog")
	If err.number <> 0 Then
		LogData "Could not create EventLog object" & err.number & err.Description
    	Exit sub
	End If

	' Log event
	oEventLog.LogEvent ITRAP_EVENT_LOG_SRC, ITRAP_EVENT_TYPE_ERROR, ITRAP_EVENT_CATEGORY, ITRAP_EVENT_ERROR_ID, sErrMsg
	Set oEventLog = Nothing
End Sub

Sub LogData( strData )
	On Error Resume Next
	Dim objLogFile
	Dim objFS

	strData = Now() & ": " & strData
	Set objFS = Server.CreateObject("Scripting.FileSystemObject")
    Set objLogFile = objFS.OpenTextFile("e:\APPS\qlscm\ESpoke\LogFileP.Txt", 8, True)
   	objLogFile.WriteLine ( strData )
   	objLogFile.Close

    	
 
End Sub
'____________________________________________________________________________________________________________
Sub Main()
	
	On Error Resume Next
	
	Dim strContract ' the name of the eHub contract
	Dim strReceipt ' receipt for the sent messsage
	Dim oESpokeObj ' the eSpoke object we are using
	Dim strMessageId 'message Id to be sent
	Dim strPayload 'Payload of message to be sent
	Dim sMsg  ' error messages
	Dim strStatusText 'the status text from eHub
	
	'LogError "Received Request", 1, ERR_SRC		
	LogData "Entering PublishESpoke.asp"
	set oESpokeObj = Server.CreateObject("ESpoke.Sender1")
	If err.number <> 0 Then
		sMsg = "ESpoke.Sender object not created  - Error Number: " & Cstr(Err.number) & " Description: " & Err.description
		LogData sMsg 
		'LogError sMsg, ERR_CREATE_ESPOKE_OBJECT, ERR_SRC
		Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    'On Error Goto 0
		Err.Raise ERR_CREATE_ESPOKE_OBJECT, ERR_SRC, sMsg
	    'Exit Sub
	End If
	LogData "Before EspokeSender1 " & Hex(err.number)&" -----" & err.description
	'LogData ("After EspokeSender1")

	' Get the data that was sent in the html
	strContract = Request("Contract")
	LogData "Contract: " & strContract
	strMessageId = Request("MessageId")
	LogData "MessageId: " & strMessageId
	strPayload = Request("Payload")
	LogData "Payload: " & strPayload
	
	Response.Write "Publishing Message " & strMessageId &" to Contract " &strContract &"<br>"
	 ' Publish the message
	 oESpokeObj.processMessage strPayload, strMessageId, strContract
	
	'LogData "Err " & err.Description 
	
	If err.number <> 0 Then
		sMsg = "Could not send payload to eHub.  Error Number: " & Hex(Err.number) &"-" & Err.number &Err.Description &" " &" Contract: "& strContract & " Message Id: " & strMessageId & " Payload: " & strPayload  _  
			 & " Description: " & Err.description
		'sMsg="Entered error code"
		LogData sMsg
	'	LogError sMsg, ERR_SEND_PAYLOAD, ERR_SRC
		Response.Write sMsg & "<br>"
		'Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_GET_PAYLOAD, ERR_SRC, sMsg
	    Exit Sub
	End If
	LogData "Message Published Successfully"
	Response.Write  "Message Published Successfully<br>"
	 ' Report the status
	 strStatusText = oESpokeObj.StatusText	
	
	If err.number <> 0 Then
		sMsg = "Did not receive StatusText Error Number: "& Hex(Err.number) & Err.number _
			& "Description: " & Err.description
		LogData sMsg
	'	LogError sMsg, ERR_SEND_PAYLOAD, ERR_SRC
		Response.Write(sMsg)
		'Response.Status = HTTP_STATUS_INTERNAL_ERROR
	    On Error Goto 0
		Err.Raise ERR_GET_PAYLOAD, ERR_SRC, sMsg
	    Exit Sub
	End If
	LogData "StatusText: " & strStatusText 
	Response.Write  "StatusText " & strStatusText & "<br>"  

	 ' Report the response
	 'strReceipt = oESpokeObj.Receipt	
	'LogData "Err " & err.Description 
	
	'If err.number <> 0 Then
	'	sMsg = "Did not receive receipt Error Number: "& Hex(Err.number) & Err.number _
	'		& "Description: " & Err.description
	'	LogData sMsg
	''	LogError sMsg, ERR_SEND_PAYLOAD, ERR_SRC
	'	Response.Write sMsg
	'	'Response.Status = HTTP_STATUS_INTERNAL_ERROR
	 '   On Error Goto 0
	'	Err.Raise ERR_GET_PAYLOAD, ERR_SRC, sMsg
	 '   Exit Sub
	'else
	'	LogData "Receipt: " & strReceipt 
	'	Response.Write "Receipt " & strReceipt  & "<br>" 
	'End If	
End Sub
'____________________________________________________________________________________________________________________________________________________________

	' call the main sub
	On Error Resume Next
'	Response.Status = HTTP_STATUS_ACCEPTED
'	Response.end

	call Main()
%>