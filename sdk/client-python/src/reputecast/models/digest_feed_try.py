from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="DigestFeedTry")



@_attrs_define
class DigestFeedTry:
    """ 
        Attributes:
            at (datetime.datetime):
            error (str):
            kind (str):
     """

    at: datetime.datetime
    error: str
    kind: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        error = self.error

        kind = self.kind


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "error": error,
            "kind": kind,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        error = d.pop("error")

        kind = d.pop("kind")

        digest_feed_try = cls(
            at=at,
            error=error,
            kind=kind,
        )

        return digest_feed_try

