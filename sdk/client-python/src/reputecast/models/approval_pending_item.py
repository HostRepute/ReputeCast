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





T = TypeVar("T", bound="ApprovalPendingItem")



@_attrs_define
class ApprovalPendingItem:
    """ 
        Attributes:
            audience (int):
            author (ApprovalPerson):
            id (str):
            name (str):
            requested_at (datetime.datetime):
     """

    audience: int
    author: ApprovalPerson
    id: str
    name: str
    requested_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        audience = self.audience

        author = self.author.to_dict()

        id = self.id

        name = self.name

        requested_at = self.requested_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "author": author,
            "id": id,
            "name": name,
            "requestedAt": requested_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        d = dict(src_dict)
        audience = d.pop("audience")

        author = ApprovalPerson.from_dict(d.pop("author"))




        id = d.pop("id")

        name = d.pop("name")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        approval_pending_item = cls(
            audience=audience,
            author=author,
            id=id,
            name=name,
            requested_at=requested_at,
        )

        return approval_pending_item

