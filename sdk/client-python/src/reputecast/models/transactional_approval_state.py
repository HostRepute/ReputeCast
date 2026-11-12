from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_approval_state_status import TransactionalApprovalStateStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalApprovalState")



@_attrs_define
class TransactionalApprovalState:
    """ 
        Attributes:
            approvers (list[str]):
            can_request (bool):
            covers_draft (bool):
            required (bool):
            status (TransactionalApprovalStateStatus):
            comment (str | Unset):
            decided_at (datetime.datetime | Unset):
            decided_by (str | Unset):
            note (str | Unset):
            request_id (str | Unset):
            request_reason (str | Unset):
            requested_at (datetime.datetime | Unset):
            requested_by (str | Unset):
     """

    approvers: list[str]
    can_request: bool
    covers_draft: bool
    required: bool
    status: TransactionalApprovalStateStatus
    comment: str | Unset = UNSET
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: str | Unset = UNSET
    note: str | Unset = UNSET
    request_id: str | Unset = UNSET
    request_reason: str | Unset = UNSET
    requested_at: datetime.datetime | Unset = UNSET
    requested_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approvers = self.approvers



        can_request = self.can_request

        covers_draft = self.covers_draft

        required = self.required

        status = self.status.value

        comment = self.comment

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by = self.decided_by

        note = self.note

        request_id = self.request_id

        request_reason = self.request_reason

        requested_at: str | Unset = UNSET
        if not isinstance(self.requested_at, Unset):
            requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "canRequest": can_request,
            "coversDraft": covers_draft,
            "required": required,
            "status": status,
        })
        if comment is not UNSET:
            field_dict["comment"] = comment
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by
        if note is not UNSET:
            field_dict["note"] = note
        if request_id is not UNSET:
            field_dict["requestId"] = request_id
        if request_reason is not UNSET:
            field_dict["requestReason"] = request_reason
        if requested_at is not UNSET:
            field_dict["requestedAt"] = requested_at
        if requested_by is not UNSET:
            field_dict["requestedBy"] = requested_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        can_request = d.pop("canRequest")

        covers_draft = d.pop("coversDraft")

        required = d.pop("required")

        status = TransactionalApprovalStateStatus(d.pop("status"))




        comment = d.pop("comment", UNSET)

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        decided_by = d.pop("decidedBy", UNSET)

        note = d.pop("note", UNSET)

        request_id = d.pop("requestId", UNSET)

        request_reason = d.pop("requestReason", UNSET)

        _requested_at = d.pop("requestedAt", UNSET)
        requested_at: datetime.datetime | Unset
        if isinstance(_requested_at,  Unset):
            requested_at = UNSET
        else:
            requested_at = datetime.datetime.fromisoformat(_requested_at)




        requested_by = d.pop("requestedBy", UNSET)

        transactional_approval_state = cls(
            approvers=approvers,
            can_request=can_request,
            covers_draft=covers_draft,
            required=required,
            status=status,
            comment=comment,
            decided_at=decided_at,
            decided_by=decided_by,
            note=note,
            request_id=request_id,
            request_reason=request_reason,
            requested_at=requested_at,
            requested_by=requested_by,
        )

        return transactional_approval_state

