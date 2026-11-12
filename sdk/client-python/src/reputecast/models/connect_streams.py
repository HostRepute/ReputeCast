from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.connect_stream_view import ConnectStreamView





T = TypeVar("T", bound="ConnectStreams")



@_attrs_define
class ConnectStreams:
    """ 
        Attributes:
            defaults (str):
            saved (bool):
            streams (list[ConnectStreamView]):
            tips (list[str]):
     """

    defaults: str
    saved: bool
    streams: list[ConnectStreamView]
    tips: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_stream_view import ConnectStreamView # noqa: PLC0415
        defaults = self.defaults

        saved = self.saved

        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)



        tips = self.tips




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaults": defaults,
            "saved": saved,
            "streams": streams,
            "tips": tips,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_stream_view import ConnectStreamView # noqa: PLC0415
        d = dict(src_dict)
        defaults = d.pop("defaults")

        saved = d.pop("saved")

        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = ConnectStreamView.from_dict(streams_item_data)



            streams.append(streams_item)


        tips = cast(list[str], d.pop("tips"))


        connect_streams = cls(
            defaults=defaults,
            saved=saved,
            streams=streams,
            tips=tips,
        )

        return connect_streams

