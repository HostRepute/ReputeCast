from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderBroadcastStreamResult")



@_attrs_define
class ProviderBroadcastStreamResult:
    """ 
        Attributes:
            stream_id (str):
     """

    stream_id: str





    def to_dict(self) -> dict[str, Any]:
        stream_id = self.stream_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "streamId": stream_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        stream_id = d.pop("streamId")

        provider_broadcast_stream_result = cls(
            stream_id=stream_id,
        )

        return provider_broadcast_stream_result

