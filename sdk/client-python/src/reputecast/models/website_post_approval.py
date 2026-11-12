from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_approval_status import WebsitePostApprovalStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsitePostApproval")



@_attrs_define
class WebsitePostApproval:
    """ 
        Attributes:
            approvers (list[str]):
            id (str):
            requested_at (datetime.datetime):
            requested_by (str):
            rule (str):
            status (WebsitePostApprovalStatus):
            note (str | Unset):
     """

    approvers: list[str]
    id: str
    requested_at: datetime.datetime
    requested_by: str
    rule: str
    status: WebsitePostApprovalStatus
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approvers = self.approvers



        id = self.id

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by

        rule = self.rule

        status = self.status.value

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "id": id,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
            "rule": rule,
            "status": status,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        id = d.pop("id")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = d.pop("requestedBy")

        rule = d.pop("rule")

        status = WebsitePostApprovalStatus(d.pop("status"))




        note = d.pop("note", UNSET)

        website_post_approval = cls(
            approvers=approvers,
            id=id,
            requested_at=requested_at,
            requested_by=requested_by,
            rule=rule,
            status=status,
            note=note,
        )

        return website_post_approval

