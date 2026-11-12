from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectSelfTest")



@_attrs_define
class ConnectSelfTest:
    """ 
        Attributes:
            at (datetime.datetime):
            ok (bool):
            private (bool):
            code (str | Unset):
            reason (str | Unset):
     """

    at: datetime.datetime
    ok: bool
    private: bool
    code: str | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        ok = self.ok

        private = self.private

        code = self.code

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "ok": ok,
            "private": private,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        ok = d.pop("ok")

        private = d.pop("private")

        code = d.pop("code", UNSET)

        reason = d.pop("reason", UNSET)

        connect_self_test = cls(
            at=at,
            ok=ok,
            private=private,
            code=code,
            reason=reason,
        )

        return connect_self_test

