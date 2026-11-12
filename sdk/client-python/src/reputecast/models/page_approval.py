from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PageApproval")



@_attrs_define
class PageApproval:
    """ 
        Attributes:
            id (str):
            requested_at (datetime.datetime):
            requested_by (str):
     """

    id: str
    requested_at: datetime.datetime
    requested_by: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = d.pop("requestedBy")

        page_approval = cls(
            id=id,
            requested_at=requested_at,
            requested_by=requested_by,
        )

        return page_approval

