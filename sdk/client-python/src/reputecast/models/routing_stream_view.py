from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_stream_view_failover import RoutingStreamViewFailover
from ..models.routing_stream_view_stream import RoutingStreamViewStream
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.routing_condition_view import RoutingConditionView
  from ..models.routing_split import RoutingSplit
  from ..models.routing_target_vet import RoutingTargetVet
  from ..models.routing_volume import RoutingVolume





T = TypeVar("T", bound="RoutingStreamView")



@_attrs_define
class RoutingStreamView:
    """ 
        Attributes:
            conditions (list[RoutingConditionView]):
            failover (RoutingStreamViewFailover):
            fallback (str):
            splits (list[RoutingSplit]):
            stream (RoutingStreamViewStream):
            version (str):
            bulk_terms_acknowledged_at (datetime.datetime | Unset):
            failover_vet (RoutingTargetVet | Unset):
            priority (bool | Unset):
            sender_domains (list[str] | Unset):
            volume (RoutingVolume | Unset):
     """

    conditions: list[RoutingConditionView]
    failover: RoutingStreamViewFailover
    fallback: str
    splits: list[RoutingSplit]
    stream: RoutingStreamViewStream
    version: str
    bulk_terms_acknowledged_at: datetime.datetime | Unset = UNSET
    failover_vet: RoutingTargetVet | Unset = UNSET
    priority: bool | Unset = UNSET
    sender_domains: list[str] | Unset = UNSET
    volume: RoutingVolume | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_condition_view import RoutingConditionView # noqa: PLC0415
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        from ..models.routing_target_vet import RoutingTargetVet # noqa: PLC0415
        from ..models.routing_volume import RoutingVolume # noqa: PLC0415
        conditions = []
        for conditions_item_data in self.conditions:
            conditions_item = conditions_item_data.to_dict()
            conditions.append(conditions_item)



        failover = self.failover.value

        fallback = self.fallback

        splits = []
        for splits_item_data in self.splits:
            splits_item = splits_item_data.to_dict()
            splits.append(splits_item)



        stream = self.stream.value

        version = self.version

        bulk_terms_acknowledged_at: str | Unset = UNSET
        if not isinstance(self.bulk_terms_acknowledged_at, Unset):
            bulk_terms_acknowledged_at = self.bulk_terms_acknowledged_at.isoformat()

        failover_vet: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failover_vet, Unset):
            failover_vet = self.failover_vet.to_dict()

        priority = self.priority

        sender_domains: list[str] | Unset = UNSET
        if not isinstance(self.sender_domains, Unset):
            sender_domains = self.sender_domains



        volume: dict[str, Any] | Unset = UNSET
        if not isinstance(self.volume, Unset):
            volume = self.volume.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "failover": failover,
            "fallback": fallback,
            "splits": splits,
            "stream": stream,
            "version": version,
        })
        if bulk_terms_acknowledged_at is not UNSET:
            field_dict["bulkTermsAcknowledgedAt"] = bulk_terms_acknowledged_at
        if failover_vet is not UNSET:
            field_dict["failoverVet"] = failover_vet
        if priority is not UNSET:
            field_dict["priority"] = priority
        if sender_domains is not UNSET:
            field_dict["senderDomains"] = sender_domains
        if volume is not UNSET:
            field_dict["volume"] = volume

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_condition_view import RoutingConditionView # noqa: PLC0415
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        from ..models.routing_target_vet import RoutingTargetVet # noqa: PLC0415
        from ..models.routing_volume import RoutingVolume # noqa: PLC0415
        d = dict(src_dict)
        conditions = []
        _conditions = d.pop("conditions")
        for conditions_item_data in (_conditions):
            conditions_item = RoutingConditionView.from_dict(conditions_item_data)



            conditions.append(conditions_item)


        failover = RoutingStreamViewFailover(d.pop("failover"))




        fallback = d.pop("fallback")

        splits = []
        _splits = d.pop("splits")
        for splits_item_data in (_splits):
            splits_item = RoutingSplit.from_dict(splits_item_data)



            splits.append(splits_item)


        stream = RoutingStreamViewStream(d.pop("stream"))




        version = d.pop("version")

        _bulk_terms_acknowledged_at = d.pop("bulkTermsAcknowledgedAt", UNSET)
        bulk_terms_acknowledged_at: datetime.datetime | Unset
        if isinstance(_bulk_terms_acknowledged_at,  Unset):
            bulk_terms_acknowledged_at = UNSET
        else:
            bulk_terms_acknowledged_at = datetime.datetime.fromisoformat(_bulk_terms_acknowledged_at)




        _failover_vet = d.pop("failoverVet", UNSET)
        failover_vet: RoutingTargetVet | Unset
        if isinstance(_failover_vet,  Unset):
            failover_vet = UNSET
        else:
            failover_vet = RoutingTargetVet.from_dict(_failover_vet)




        priority = d.pop("priority", UNSET)

        sender_domains = cast(list[str], d.pop("senderDomains", UNSET))


        _volume = d.pop("volume", UNSET)
        volume: RoutingVolume | Unset
        if isinstance(_volume,  Unset):
            volume = UNSET
        else:
            volume = RoutingVolume.from_dict(_volume)




        routing_stream_view = cls(
            conditions=conditions,
            failover=failover,
            fallback=fallback,
            splits=splits,
            stream=stream,
            version=version,
            bulk_terms_acknowledged_at=bulk_terms_acknowledged_at,
            failover_vet=failover_vet,
            priority=priority,
            sender_domains=sender_domains,
            volume=volume,
        )

        return routing_stream_view

