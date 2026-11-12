from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_pool_health_only_stream import DeliverabilityPoolHealthOnlyStream
from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityPoolHealth")



@_attrs_define
class DeliverabilityPoolHealth:
    """ 
        Attributes:
            account_status (str):
            adapter (str):
            managed_reputation (bool):
            provider (str):
            provider_id (str):
            short (str):
            status (str):
            bounce_rate (float | Unset):
            complaint_rate (float | Unset):
            dedicated_ip_option (str | Unset):
            only_stream (DeliverabilityPoolHealthOnlyStream | Unset):
            region (str | Unset):
            volume_this_month (int | Unset):
     """

    account_status: str
    adapter: str
    managed_reputation: bool
    provider: str
    provider_id: str
    short: str
    status: str
    bounce_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    dedicated_ip_option: str | Unset = UNSET
    only_stream: DeliverabilityPoolHealthOnlyStream | Unset = UNSET
    region: str | Unset = UNSET
    volume_this_month: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        account_status = self.account_status

        adapter = self.adapter

        managed_reputation = self.managed_reputation

        provider = self.provider

        provider_id = self.provider_id

        short = self.short

        status = self.status

        bounce_rate = self.bounce_rate

        complaint_rate = self.complaint_rate

        dedicated_ip_option = self.dedicated_ip_option

        only_stream: str | Unset = UNSET
        if not isinstance(self.only_stream, Unset):
            only_stream = self.only_stream.value


        region = self.region

        volume_this_month = self.volume_this_month


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accountStatus": account_status,
            "adapter": adapter,
            "managedReputation": managed_reputation,
            "provider": provider,
            "providerId": provider_id,
            "short": short,
            "status": status,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if dedicated_ip_option is not UNSET:
            field_dict["dedicatedIpOption"] = dedicated_ip_option
        if only_stream is not UNSET:
            field_dict["onlyStream"] = only_stream
        if region is not UNSET:
            field_dict["region"] = region
        if volume_this_month is not UNSET:
            field_dict["volumeThisMonth"] = volume_this_month

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        account_status = d.pop("accountStatus")

        adapter = d.pop("adapter")

        managed_reputation = d.pop("managedReputation")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        status = d.pop("status")

        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        dedicated_ip_option = d.pop("dedicatedIpOption", UNSET)

        _only_stream = d.pop("onlyStream", UNSET)
        only_stream: DeliverabilityPoolHealthOnlyStream | Unset
        if isinstance(_only_stream,  Unset):
            only_stream = UNSET
        else:
            only_stream = DeliverabilityPoolHealthOnlyStream(_only_stream)




        region = d.pop("region", UNSET)

        volume_this_month = d.pop("volumeThisMonth", UNSET)

        deliverability_pool_health = cls(
            account_status=account_status,
            adapter=adapter,
            managed_reputation=managed_reputation,
            provider=provider,
            provider_id=provider_id,
            short=short,
            status=status,
            bounce_rate=bounce_rate,
            complaint_rate=complaint_rate,
            dedicated_ip_option=dedicated_ip_option,
            only_stream=only_stream,
            region=region,
            volume_this_month=volume_this_month,
        )

        return deliverability_pool_health

