from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnReach")



@_attrs_define
class EarnReach:
    """ 
        Attributes:
            checked (bool):
            host (str):
            ok (bool):
            at (datetime.datetime | Unset):
            code (str | Unset):
            reason (str | Unset):
     """

    checked: bool
    host: str
    ok: bool
    at: datetime.datetime | Unset = UNSET
    code: str | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checked = self.checked

        host = self.host

        ok = self.ok

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        code = self.code

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "host": host,
            "ok": ok,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if code is not UNSET:
            field_dict["code"] = code
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked = d.pop("checked")

        host = d.pop("host")

        ok = d.pop("ok")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        code = d.pop("code", UNSET)

        reason = d.pop("reason", UNSET)

        earn_reach = cls(
            checked=checked,
            host=host,
            ok=ok,
            at=at,
            code=code,
            reason=reason,
        )

        return earn_reach

