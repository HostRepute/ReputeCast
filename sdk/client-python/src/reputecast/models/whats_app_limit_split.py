from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppLimitSplit")



@_attrs_define
class WhatsAppLimitSplit:
    """ 
        Attributes:
            later (int):
            left (int):
            limit (int):
            now (int):
            used (int):
            later_at (datetime.datetime | Unset):
            used_by (str | Unset):
            used_by_at (datetime.datetime | Unset):
            used_by_n (int | Unset):
     """

    later: int
    left: int
    limit: int
    now: int
    used: int
    later_at: datetime.datetime | Unset = UNSET
    used_by: str | Unset = UNSET
    used_by_at: datetime.datetime | Unset = UNSET
    used_by_n: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        later = self.later

        left = self.left

        limit = self.limit

        now = self.now

        used = self.used

        later_at: str | Unset = UNSET
        if not isinstance(self.later_at, Unset):
            later_at = self.later_at.isoformat()

        used_by = self.used_by

        used_by_at: str | Unset = UNSET
        if not isinstance(self.used_by_at, Unset):
            used_by_at = self.used_by_at.isoformat()

        used_by_n = self.used_by_n


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "later": later,
            "left": left,
            "limit": limit,
            "now": now,
            "used": used,
        })
        if later_at is not UNSET:
            field_dict["laterAt"] = later_at
        if used_by is not UNSET:
            field_dict["usedBy"] = used_by
        if used_by_at is not UNSET:
            field_dict["usedByAt"] = used_by_at
        if used_by_n is not UNSET:
            field_dict["usedByN"] = used_by_n

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        later = d.pop("later")

        left = d.pop("left")

        limit = d.pop("limit")

        now = d.pop("now")

        used = d.pop("used")

        _later_at = d.pop("laterAt", UNSET)
        later_at: datetime.datetime | Unset
        if isinstance(_later_at,  Unset):
            later_at = UNSET
        else:
            later_at = datetime.datetime.fromisoformat(_later_at)




        used_by = d.pop("usedBy", UNSET)

        _used_by_at = d.pop("usedByAt", UNSET)
        used_by_at: datetime.datetime | Unset
        if isinstance(_used_by_at,  Unset):
            used_by_at = UNSET
        else:
            used_by_at = datetime.datetime.fromisoformat(_used_by_at)




        used_by_n = d.pop("usedByN", UNSET)

        whats_app_limit_split = cls(
            later=later,
            left=left,
            limit=limit,
            now=now,
            used=used,
            later_at=later_at,
            used_by=used_by,
            used_by_at=used_by_at,
            used_by_n=used_by_n,
        )

        return whats_app_limit_split

