from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_issue_ref_status import DigestIssueRefStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestIssueRef")



@_attrs_define
class DigestIssueRef:
    """ 
        Attributes:
            id (str):
            needs_approval (bool):
            number (int):
            posts (int):
            status (DigestIssueRefStatus):
            human_opens (float | Unset):
            human_opens_delta (float | Unset):
            send_at (datetime.datetime | Unset):
     """

    id: str
    needs_approval: bool
    number: int
    posts: int
    status: DigestIssueRefStatus
    human_opens: float | Unset = UNSET
    human_opens_delta: float | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        needs_approval = self.needs_approval

        number = self.number

        posts = self.posts

        status = self.status.value

        human_opens = self.human_opens

        human_opens_delta = self.human_opens_delta

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "needsApproval": needs_approval,
            "number": number,
            "posts": posts,
            "status": status,
        })
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens
        if human_opens_delta is not UNSET:
            field_dict["humanOpensDelta"] = human_opens_delta
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        needs_approval = d.pop("needsApproval")

        number = d.pop("number")

        posts = d.pop("posts")

        status = DigestIssueRefStatus(d.pop("status"))




        human_opens = d.pop("humanOpens", UNSET)

        human_opens_delta = d.pop("humanOpensDelta", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        digest_issue_ref = cls(
            id=id,
            needs_approval=needs_approval,
            number=number,
            posts=posts,
            status=status,
            human_opens=human_opens,
            human_opens_delta=human_opens_delta,
            send_at=send_at,
        )

        return digest_issue_ref

