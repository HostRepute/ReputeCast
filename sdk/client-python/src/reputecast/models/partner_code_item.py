from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PartnerCodeItem")



@_attrs_define
class PartnerCodeItem:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            daily_limit (int):
            expires_at (datetime.datetime):
            id (str):
            label (str):
            list_name (str):
     """

    created_at: datetime.datetime
    created_by: str
    daily_limit: int
    expires_at: datetime.datetime
    id: str
    label: str
    list_name: str





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        daily_limit = self.daily_limit

        expires_at = self.expires_at.isoformat()

        id = self.id

        label = self.label

        list_name = self.list_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "dailyLimit": daily_limit,
            "expiresAt": expires_at,
            "id": id,
            "label": label,
            "listName": list_name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        daily_limit = d.pop("dailyLimit")

        expires_at = datetime.datetime.fromisoformat(d.pop("expiresAt"))




        id = d.pop("id")

        label = d.pop("label")

        list_name = d.pop("listName")

        partner_code_item = cls(
            created_at=created_at,
            created_by=created_by,
            daily_limit=daily_limit,
            expires_at=expires_at,
            id=id,
            label=label,
            list_name=list_name,
        )

        return partner_code_item

