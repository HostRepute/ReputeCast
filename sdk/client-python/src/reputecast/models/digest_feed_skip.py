from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="DigestFeedSkip")



@_attrs_define
class DigestFeedSkip:
    """ 
        Attributes:
            kind (str):
            send_at (datetime.datetime):
     """

    kind: str
    send_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "sendAt": send_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        send_at = datetime.datetime.fromisoformat(d.pop("sendAt"))




        digest_feed_skip = cls(
            kind=kind,
            send_at=send_at,
        )

        return digest_feed_skip

