from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_stream_input import RoutingStreamInput





T = TypeVar("T", bound="RoutingSaveInput")



@_attrs_define
class RoutingSaveInput:
    """ 
        Attributes:
            streams (list[RoutingStreamInput]):
            acknowledge_bulk_terms (bool | Unset):
     """

    streams: list[RoutingStreamInput]
    acknowledge_bulk_terms: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_stream_input import RoutingStreamInput # noqa: PLC0415
        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)



        acknowledge_bulk_terms = self.acknowledge_bulk_terms


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "streams": streams,
        })
        if acknowledge_bulk_terms is not UNSET:
            field_dict["acknowledgeBulkTerms"] = acknowledge_bulk_terms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_stream_input import RoutingStreamInput # noqa: PLC0415
        d = dict(src_dict)
        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = RoutingStreamInput.from_dict(streams_item_data)



            streams.append(streams_item)


        acknowledge_bulk_terms = d.pop("acknowledgeBulkTerms", UNSET)

        routing_save_input = cls(
            streams=streams,
            acknowledge_bulk_terms=acknowledge_bulk_terms,
        )

        return routing_save_input

