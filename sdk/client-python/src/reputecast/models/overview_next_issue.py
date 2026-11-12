from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_next_issue_status import OverviewNextIssueStatus
from typing import cast
import datetime






T = TypeVar("T", bound="OverviewNextIssue")



@_attrs_define
class OverviewNextIssue:
    """ 
        Attributes:
            digest_id (str):
            digest_name (str):
            needs_approval (bool):
            number (int):
            posts (int):
            send_at (datetime.datetime):
            status (OverviewNextIssueStatus):
            timezone (str):
     """

    digest_id: str
    digest_name: str
    needs_approval: bool
    number: int
    posts: int
    send_at: datetime.datetime
    status: OverviewNextIssueStatus
    timezone: str





    def to_dict(self) -> dict[str, Any]:
        digest_id = self.digest_id

        digest_name = self.digest_name

        needs_approval = self.needs_approval

        number = self.number

        posts = self.posts

        send_at = self.send_at.isoformat()

        status = self.status.value

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "digestId": digest_id,
            "digestName": digest_name,
            "needsApproval": needs_approval,
            "number": number,
            "posts": posts,
            "sendAt": send_at,
            "status": status,
            "timezone": timezone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        digest_id = d.pop("digestId")

        digest_name = d.pop("digestName")

        needs_approval = d.pop("needsApproval")

        number = d.pop("number")

        posts = d.pop("posts")

        send_at = datetime.datetime.fromisoformat(d.pop("sendAt"))




        status = OverviewNextIssueStatus(d.pop("status"))




        timezone = d.pop("timezone")

        overview_next_issue = cls(
            digest_id=digest_id,
            digest_name=digest_name,
            needs_approval=needs_approval,
            number=number,
            posts=posts,
            send_at=send_at,
            status=status,
            timezone=timezone,
        )

        return overview_next_issue

