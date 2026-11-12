from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="AITestView")



@_attrs_define
class AITestView:
    """ 
        Attributes:
            at (datetime.datetime):
            ms (int):
            ok (bool):
     """

    at: datetime.datetime
    ms: int
    ok: bool





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        ms = self.ms

        ok = self.ok


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "ms": ms,
            "ok": ok,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        ms = d.pop("ms")

        ok = d.pop("ok")

        ai_test_view = cls(
            at=at,
            ms=ms,
            ok=ok,
        )

        return ai_test_view

