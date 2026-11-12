from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.connect_stream_choice_input import ConnectStreamChoiceInput





T = TypeVar("T", bound="ConnectStreamsInput")



@_attrs_define
class ConnectStreamsInput:
    """ 
        Attributes:
            streams (list[ConnectStreamChoiceInput]):
     """

    streams: list[ConnectStreamChoiceInput]





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_stream_choice_input import ConnectStreamChoiceInput # noqa: PLC0415
        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "streams": streams,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_stream_choice_input import ConnectStreamChoiceInput # noqa: PLC0415
        d = dict(src_dict)
        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = ConnectStreamChoiceInput.from_dict(streams_item_data)



            streams.append(streams_item)


        connect_streams_input = cls(
            streams=streams,
        )

        return connect_streams_input

