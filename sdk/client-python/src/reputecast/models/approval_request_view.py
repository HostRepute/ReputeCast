from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_request_view_status import ApprovalRequestViewStatus
from ..models.approval_request_view_subject_type import ApprovalRequestViewSubjectType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.approval_audit_line import ApprovalAuditLine
  from ..models.approval_change import ApprovalChange
  from ..models.approval_decision_view import ApprovalDecisionView
  from ..models.approval_person import ApprovalPerson
  from ..models.approval_rule_match import ApprovalRuleMatch
  from ..models.approval_snapshot_summary import ApprovalSnapshotSummary





T = TypeVar("T", bound="ApprovalRequestView")



@_attrs_define
class ApprovalRequestView:
    """ 
        Attributes:
            changes (list[ApprovalChange]):
            decisions (list[ApprovalDecisionView]):
            given (int):
            id (str):
            needed (int):
            note (str):
            requested_at (datetime.datetime):
            requested_by (ApprovalPerson):
            rules (list[ApprovalRuleMatch]):
            snapshot (ApprovalSnapshotSummary):
            status (ApprovalRequestViewStatus):
            subject_type (ApprovalRequestViewSubjectType):
            version (int):
            audit (ApprovalAuditLine | Unset):
            bypass_reason (str | Unset):
            comment (str | Unset):
            decided_at (datetime.datetime | Unset):
            decided_by (str | Unset):
            follows_id (str | Unset):
            invalidated_at (datetime.datetime | Unset):
            invalidated_by (str | Unset):
            invalidated_fields (str | Unset):
            invalidated_reason (str | Unset):
     """

    changes: list[ApprovalChange]
    decisions: list[ApprovalDecisionView]
    given: int
    id: str
    needed: int
    note: str
    requested_at: datetime.datetime
    requested_by: ApprovalPerson
    rules: list[ApprovalRuleMatch]
    snapshot: ApprovalSnapshotSummary
    status: ApprovalRequestViewStatus
    subject_type: ApprovalRequestViewSubjectType
    version: int
    audit: ApprovalAuditLine | Unset = UNSET
    bypass_reason: str | Unset = UNSET
    comment: str | Unset = UNSET
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: str | Unset = UNSET
    follows_id: str | Unset = UNSET
    invalidated_at: datetime.datetime | Unset = UNSET
    invalidated_by: str | Unset = UNSET
    invalidated_fields: str | Unset = UNSET
    invalidated_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_audit_line import ApprovalAuditLine # noqa: PLC0415
        from ..models.approval_change import ApprovalChange # noqa: PLC0415
        from ..models.approval_decision_view import ApprovalDecisionView # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        from ..models.approval_snapshot_summary import ApprovalSnapshotSummary # noqa: PLC0415
        changes = []
        for changes_item_data in self.changes:
            changes_item = changes_item_data.to_dict()
            changes.append(changes_item)



        decisions = []
        for decisions_item_data in self.decisions:
            decisions_item = decisions_item_data.to_dict()
            decisions.append(decisions_item)



        given = self.given

        id = self.id

        needed = self.needed

        note = self.note

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by.to_dict()

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)



        snapshot = self.snapshot.to_dict()

        status = self.status.value

        subject_type = self.subject_type.value

        version = self.version

        audit: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audit, Unset):
            audit = self.audit.to_dict()

        bypass_reason = self.bypass_reason

        comment = self.comment

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by = self.decided_by

        follows_id = self.follows_id

        invalidated_at: str | Unset = UNSET
        if not isinstance(self.invalidated_at, Unset):
            invalidated_at = self.invalidated_at.isoformat()

        invalidated_by = self.invalidated_by

        invalidated_fields = self.invalidated_fields

        invalidated_reason = self.invalidated_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "decisions": decisions,
            "given": given,
            "id": id,
            "needed": needed,
            "note": note,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
            "rules": rules,
            "snapshot": snapshot,
            "status": status,
            "subjectType": subject_type,
            "version": version,
        })
        if audit is not UNSET:
            field_dict["audit"] = audit
        if bypass_reason is not UNSET:
            field_dict["bypassReason"] = bypass_reason
        if comment is not UNSET:
            field_dict["comment"] = comment
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by
        if follows_id is not UNSET:
            field_dict["followsId"] = follows_id
        if invalidated_at is not UNSET:
            field_dict["invalidatedAt"] = invalidated_at
        if invalidated_by is not UNSET:
            field_dict["invalidatedBy"] = invalidated_by
        if invalidated_fields is not UNSET:
            field_dict["invalidatedFields"] = invalidated_fields
        if invalidated_reason is not UNSET:
            field_dict["invalidatedReason"] = invalidated_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_audit_line import ApprovalAuditLine # noqa: PLC0415
        from ..models.approval_change import ApprovalChange # noqa: PLC0415
        from ..models.approval_decision_view import ApprovalDecisionView # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        from ..models.approval_snapshot_summary import ApprovalSnapshotSummary # noqa: PLC0415
        d = dict(src_dict)
        changes = []
        _changes = d.pop("changes")
        for changes_item_data in (_changes):
            changes_item = ApprovalChange.from_dict(changes_item_data)



            changes.append(changes_item)


        decisions = []
        _decisions = d.pop("decisions")
        for decisions_item_data in (_decisions):
            decisions_item = ApprovalDecisionView.from_dict(decisions_item_data)



            decisions.append(decisions_item)


        given = d.pop("given")

        id = d.pop("id")

        needed = d.pop("needed")

        note = d.pop("note")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = ApprovalPerson.from_dict(d.pop("requestedBy"))




        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = ApprovalRuleMatch.from_dict(rules_item_data)



            rules.append(rules_item)


        snapshot = ApprovalSnapshotSummary.from_dict(d.pop("snapshot"))




        status = ApprovalRequestViewStatus(d.pop("status"))




        subject_type = ApprovalRequestViewSubjectType(d.pop("subjectType"))




        version = d.pop("version")

        _audit = d.pop("audit", UNSET)
        audit: ApprovalAuditLine | Unset
        if isinstance(_audit,  Unset):
            audit = UNSET
        else:
            audit = ApprovalAuditLine.from_dict(_audit)




        bypass_reason = d.pop("bypassReason", UNSET)

        comment = d.pop("comment", UNSET)

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        decided_by = d.pop("decidedBy", UNSET)

        follows_id = d.pop("followsId", UNSET)

        _invalidated_at = d.pop("invalidatedAt", UNSET)
        invalidated_at: datetime.datetime | Unset
        if isinstance(_invalidated_at,  Unset):
            invalidated_at = UNSET
        else:
            invalidated_at = datetime.datetime.fromisoformat(_invalidated_at)




        invalidated_by = d.pop("invalidatedBy", UNSET)

        invalidated_fields = d.pop("invalidatedFields", UNSET)

        invalidated_reason = d.pop("invalidatedReason", UNSET)

        approval_request_view = cls(
            changes=changes,
            decisions=decisions,
            given=given,
            id=id,
            needed=needed,
            note=note,
            requested_at=requested_at,
            requested_by=requested_by,
            rules=rules,
            snapshot=snapshot,
            status=status,
            subject_type=subject_type,
            version=version,
            audit=audit,
            bypass_reason=bypass_reason,
            comment=comment,
            decided_at=decided_at,
            decided_by=decided_by,
            follows_id=follows_id,
            invalidated_at=invalidated_at,
            invalidated_by=invalidated_by,
            invalidated_fields=invalidated_fields,
            invalidated_reason=invalidated_reason,
        )

        return approval_request_view

