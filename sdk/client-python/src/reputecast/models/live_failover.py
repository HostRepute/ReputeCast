from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveFailover")



@_attrs_define
class LiveFailover:
    """ 
        Attributes:
            stream (str):
            to (str):
     """

    stream: str
    to: str





    def to_dict(self) -> dict[str, Any]:
        stream = self.stream

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "stream": stream,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        stream = d.pop("stream")

        to = d.pop("to")

        live_failover = cls(
            stream=stream,
            to=to,
        )

        return live_failover

