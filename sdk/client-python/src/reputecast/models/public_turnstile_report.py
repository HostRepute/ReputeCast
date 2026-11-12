from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PublicTurnstileReport")



@_attrs_define
class PublicTurnstileReport:
    """ 
        Attributes:
            code (str): The widget's error code Example: 110200.
            count (int):
            error (str): What the code means, in the reader's language
            last_at (datetime.datetime):
     """

    code: str
    count: int
    error: str
    last_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        count = self.count

        error = self.error

        last_at = self.last_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "count": count,
            "error": error,
            "lastAt": last_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        count = d.pop("count")

        error = d.pop("error")

        last_at = datetime.datetime.fromisoformat(d.pop("lastAt"))




        public_turnstile_report = cls(
            code=code,
            count=count,
            error=error,
            last_at=last_at,
        )

        return public_turnstile_report

