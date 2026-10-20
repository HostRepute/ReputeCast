package agency

// Server text people read (problems, reasons), in messages/<locale>.json.
// Audit entries keep their English.
var (
	mAgencyOff   = msgs.M("agencyOff")
	mNotStaff    = msgs.M("notStaff")
	mDenied      = msgs.M("denied")
	mDeniedAsk   = msgs.M("deniedAsk")
	mWhatView    = msgs.M("what.view")
	mWhatUsage   = msgs.M("what.usage")
	mWhatLibrary = msgs.M("what.library")
	mWhatManage  = msgs.M("what.manage")
	mAsk         = msgs.M("ask")

	// Staff and assignments.
	mNotYoursOwner     = msgs.M("notYoursOwner")
	mNotYours          = msgs.M("notYours")
	mLevelPick         = msgs.M("levelPick")
	mPickAccount       = msgs.M("pickAccount")
	mEmailInvalid      = msgs.M("emailInvalid")
	mClientPerson      = msgs.M("clientPerson")
	mAlreadyAdmin      = msgs.M("alreadyAdmin")
	mAlreadyStaff      = msgs.M("alreadyStaff")
	mAssignmentInvalid = msgs.M("assignmentInvalid")
	mStaffClientRole   = msgs.M("staffClientRole")
	mAdminLevel        = msgs.M("adminLevel")
	mAdminRemove       = msgs.M("adminRemove")
	mRemoveSelf        = msgs.M("removeSelf")
	mEscalation        = msgs.M("escalation")
	mEscalationNoRole  = msgs.M("escalationNoRole")
	mNoAccountYet      = msgs.M("noAccountYet")
	mLastOwner         = msgs.M("lastOwner")
	mOutranks          = msgs.M("outranks")
	mUnassignSelf      = msgs.M("unassignSelf")

	// Agency mode.
	mPickWorkspace   = msgs.M("pickWorkspace")
	mPickTemplate    = msgs.M("pickTemplate")
	mAgencyNotClient = msgs.M("agencyNotClient")

	// Usage and clients.
	mMonthInvalid = msgs.M("monthInvalid")
	mPickStaff    = msgs.M("pickStaff")

	// Workspace templates.
	mNotAgencyWorkspace = msgs.M("notAgencyWorkspace")
	mTemplateName       = msgs.M("templateName")
	mTemplateNameTaken  = msgs.M("templateNameTaken")

	// Library, clients and search.
	mCopyNeeds             = msgs.M("copyNeeds")
	mCopyNeedsHere         = msgs.M("copyNeedsHere")
	mSMSFromComposer       = msgs.M("smsFromComposer")
	mFlowNotInPlace        = msgs.M("flowNotInPlace")
	mKindMismatch          = msgs.M("kindMismatch")
	mLibraryNameTaken      = msgs.M("libraryNameTaken")
	mEmptyItem             = msgs.M("emptyItem")
	mCopyGone              = msgs.M("copyGone")
	mClientName            = msgs.M("clientName")
	mSlugInvalid           = msgs.M("slugInvalid")
	mSlugTaken             = msgs.M("slugTaken")
	mTimezone              = msgs.M("timezone")
	mOwnerManager          = msgs.M("ownerManager")
	mFirstPersonRole       = msgs.M("firstPersonRole")
	mAgencyWorkspaceDelete = msgs.M("agencyWorkspaceDelete")
	mSearchShort           = msgs.M("searchShort")
)
