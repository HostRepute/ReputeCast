from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_stream_route_failover import ProviderStreamRouteFailover
from ..models.provider_stream_route_stream import ProviderStreamRouteStream
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.provider_route_share import ProviderRouteShare





T = TypeVar("T", bound="ProviderStreamRoute")



@_attrs_define
class ProviderStreamRoute:
    """ 
        Attributes:
            conditions (int):
            failover (ProviderStreamRouteFailover):
            splits (list[ProviderRouteShare]):
            stream (ProviderStreamRouteStream):
            fallback (ProviderRouteShare | Unset):
     """

    conditions: int
    failover: ProviderStreamRouteFailover
    splits: list[ProviderRouteShare]
    stream: ProviderStreamRouteStream
    fallback: ProviderRouteShare | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_route_share import ProviderRouteShare # noqa: PLC0415
        conditions = self.conditions

        failover = self.failover.value

        splits = []
        for splits_item_data in self.splits:
            splits_item = splits_item_data.to_dict()
            splits.append(splits_item)



        stream = self.stream.value

        fallback: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fallback, Unset):
            fallback = self.fallback.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "failover": failover,
            "splits": splits,
            "stream": stream,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_route_share import ProviderRouteShare # noqa: PLC0415
        d = dict(src_dict)
        conditions = d.pop("conditions")

        failover = ProviderStreamRouteFailover(d.pop("failover"))




        splits = []
        _splits = d.pop("splits")
        for splits_item_data in (_splits):
            splits_item = ProviderRouteShare.from_dict(splits_item_data)



            splits.append(splits_item)


        stream = ProviderStreamRouteStream(d.pop("stream"))




        _fallback = d.pop("fallback", UNSET)
        fallback: ProviderRouteShare | Unset
        if isinstance(_fallback,  Unset):
            fallback = UNSET
        else:
            fallback = ProviderRouteShare.from_dict(_fallback)




        provider_stream_route = cls(
            conditions=conditions,
            failover=failover,
            splits=splits,
            stream=stream,
            fallback=fallback,
        )

        return provider_stream_route

