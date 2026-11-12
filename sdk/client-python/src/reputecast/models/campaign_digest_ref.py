from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_digest_ref_issue_status import CampaignDigestRefIssueStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CampaignDigestRef")



@_attrs_define
class CampaignDigestRef:
    """ 
        Attributes:
            id (str):
            issue_status (CampaignDigestRefIssueStatus):
            name (str):
            posts (int):
            approval_id (str | Unset):
            prepared_at (datetime.datetime | Unset):
            send_at (datetime.datetime | Unset):
     """

    id: str
    issue_status: CampaignDigestRefIssueStatus
    name: str
    posts: int
    approval_id: str | Unset = UNSET
    prepared_at: datetime.datetime | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        issue_status = self.issue_status.value

        name = self.name

        posts = self.posts

        approval_id = self.approval_id

        prepared_at: str | Unset = UNSET
        if not isinstance(self.prepared_at, Unset):
            prepared_at = self.prepared_at.isoformat()

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "issueStatus": issue_status,
            "name": name,
            "posts": posts,
        })
        if approval_id is not UNSET:
            field_dict["approvalId"] = approval_id
        if prepared_at is not UNSET:
            field_dict["preparedAt"] = prepared_at
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        issue_status = CampaignDigestRefIssueStatus(d.pop("issueStatus"))




        name = d.pop("name")

        posts = d.pop("posts")

        approval_id = d.pop("approvalId", UNSET)

        _prepared_at = d.pop("preparedAt", UNSET)
        prepared_at: datetime.datetime | Unset
        if isinstance(_prepared_at,  Unset):
            prepared_at = UNSET
        else:
            prepared_at = datetime.datetime.fromisoformat(_prepared_at)




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        campaign_digest_ref = cls(
            id=id,
            issue_status=issue_status,
            name=name,
            posts=posts,
            approval_id=approval_id,
            prepared_at=prepared_at,
            send_at=send_at,
        )

        return campaign_digest_ref

