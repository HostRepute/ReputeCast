from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.approval_person import ApprovalPerson





T = TypeVar("T", bound="ApprovalCommentView")



@_attrs_define
class ApprovalCommentView:
    """ 
        Attributes:
            at (datetime.datetime):
            author (ApprovalPerson):
            body (str):
            id (str):
     """

    at: datetime.datetime
    author: ApprovalPerson
    body: str
    id: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        at = self.at.isoformat()

        author = self.author.to_dict()

        body = self.body

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "author": author,
            "body": body,
            "id": id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        author = ApprovalPerson.from_dict(d.pop("author"))




        body = d.pop("body")

        id = d.pop("id")

        approval_comment_view = cls(
            at=at,
            author=author,
            body=body,
            id=id,
        )

        return approval_comment_view

