from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_guidance_cause import PlacementGuidanceCause
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_auth_readiness import PlacementAuthReadiness
  from ..models.placement_auth_summary import PlacementAuthSummary
  from ..models.placement_earlier import PlacementEarlier
  from ..models.placement_other import PlacementOther





T = TypeVar("T", bound="PlacementGuidance")



@_attrs_define
class PlacementGuidance:
    """ 
        Attributes:
            at (list[str]):
            auth (PlacementAuthSummary):
            auth_providers (int):
            cause (PlacementGuidanceCause):
            copies (int):
            host_repute (bool):
            provider_id (str):
            provider_name (str):
            spam (int):
            status (str):
            earlier (PlacementEarlier | Unset):
            ip (str | Unset):
            other (PlacementOther | Unset):
            readiness (PlacementAuthReadiness | Unset):
            warm_day (int | Unset):
            warm_days (int | Unset):
     """

    at: list[str]
    auth: PlacementAuthSummary
    auth_providers: int
    cause: PlacementGuidanceCause
    copies: int
    host_repute: bool
    provider_id: str
    provider_name: str
    spam: int
    status: str
    earlier: PlacementEarlier | Unset = UNSET
    ip: str | Unset = UNSET
    other: PlacementOther | Unset = UNSET
    readiness: PlacementAuthReadiness | Unset = UNSET
    warm_day: int | Unset = UNSET
    warm_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_auth_readiness import PlacementAuthReadiness # noqa: PLC0415
        from ..models.placement_auth_summary import PlacementAuthSummary # noqa: PLC0415
        from ..models.placement_earlier import PlacementEarlier # noqa: PLC0415
        from ..models.placement_other import PlacementOther # noqa: PLC0415
        at = self.at



        auth = self.auth.to_dict()

        auth_providers = self.auth_providers

        cause = self.cause.value

        copies = self.copies

        host_repute = self.host_repute

        provider_id = self.provider_id

        provider_name = self.provider_name

        spam = self.spam

        status = self.status

        earlier: dict[str, Any] | Unset = UNSET
        if not isinstance(self.earlier, Unset):
            earlier = self.earlier.to_dict()

        ip = self.ip

        other: dict[str, Any] | Unset = UNSET
        if not isinstance(self.other, Unset):
            other = self.other.to_dict()

        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()

        warm_day = self.warm_day

        warm_days = self.warm_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "auth": auth,
            "authProviders": auth_providers,
            "cause": cause,
            "copies": copies,
            "hostRepute": host_repute,
            "providerId": provider_id,
            "providerName": provider_name,
            "spam": spam,
            "status": status,
        })
        if earlier is not UNSET:
            field_dict["earlier"] = earlier
        if ip is not UNSET:
            field_dict["ip"] = ip
        if other is not UNSET:
            field_dict["other"] = other
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if warm_day is not UNSET:
            field_dict["warmDay"] = warm_day
        if warm_days is not UNSET:
            field_dict["warmDays"] = warm_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_auth_readiness import PlacementAuthReadiness # noqa: PLC0415
        from ..models.placement_auth_summary import PlacementAuthSummary # noqa: PLC0415
        from ..models.placement_earlier import PlacementEarlier # noqa: PLC0415
        from ..models.placement_other import PlacementOther # noqa: PLC0415
        d = dict(src_dict)
        at = cast(list[str], d.pop("at"))


        auth = PlacementAuthSummary.from_dict(d.pop("auth"))




        auth_providers = d.pop("authProviders")

        cause = PlacementGuidanceCause(d.pop("cause"))




        copies = d.pop("copies")

        host_repute = d.pop("hostRepute")

        provider_id = d.pop("providerId")

        provider_name = d.pop("providerName")

        spam = d.pop("spam")

        status = d.pop("status")

        _earlier = d.pop("earlier", UNSET)
        earlier: PlacementEarlier | Unset
        if isinstance(_earlier,  Unset):
            earlier = UNSET
        else:
            earlier = PlacementEarlier.from_dict(_earlier)




        ip = d.pop("ip", UNSET)

        _other = d.pop("other", UNSET)
        other: PlacementOther | Unset
        if isinstance(_other,  Unset):
            other = UNSET
        else:
            other = PlacementOther.from_dict(_other)




        _readiness = d.pop("readiness", UNSET)
        readiness: PlacementAuthReadiness | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = PlacementAuthReadiness.from_dict(_readiness)




        warm_day = d.pop("warmDay", UNSET)

        warm_days = d.pop("warmDays", UNSET)

        placement_guidance = cls(
            at=at,
            auth=auth,
            auth_providers=auth_providers,
            cause=cause,
            copies=copies,
            host_repute=host_repute,
            provider_id=provider_id,
            provider_name=provider_name,
            spam=spam,
            status=status,
            earlier=earlier,
            ip=ip,
            other=other,
            readiness=readiness,
            warm_day=warm_day,
            warm_days=warm_days,
        )

        return placement_guidance

