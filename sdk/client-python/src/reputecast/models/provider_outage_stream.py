from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_outage_stream_policy import ProviderOutageStreamPolicy
from ..models.provider_outage_stream_role import ProviderOutageStreamRole
from ..models.provider_outage_stream_state import ProviderOutageStreamState
from ..models.provider_outage_stream_stream import ProviderOutageStreamStream
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_failover_alternative import ProviderFailoverAlternative
  from ..models.provider_held_campaign import ProviderHeldCampaign
  from ..models.provider_named_count import ProviderNamedCount
  from ..models.vet_check import VetCheck





T = TypeVar("T", bound="ProviderOutageStream")



@_attrs_define
class ProviderOutageStream:
    """ 
        Attributes:
            alternatives (list[ProviderFailoverAlternative]):
            fallback_checks (list[VetCheck]):
            policy (ProviderOutageStreamPolicy):
            rerouted (int):
            rerouted_by (list[ProviderNamedCount]):
            role (ProviderOutageStreamRole):
            since (datetime.datetime):
            state (ProviderOutageStreamState):
            stream (ProviderOutageStreamStream):
            switch_back_minutes (int):
            campaign (ProviderHeldCampaign | Unset):
            fallback (str | Unset):
            fallback_short (str | Unset):
            primary (str | Unset):
            suggestion (str | Unset):
     """

    alternatives: list[ProviderFailoverAlternative]
    fallback_checks: list[VetCheck]
    policy: ProviderOutageStreamPolicy
    rerouted: int
    rerouted_by: list[ProviderNamedCount]
    role: ProviderOutageStreamRole
    since: datetime.datetime
    state: ProviderOutageStreamState
    stream: ProviderOutageStreamStream
    switch_back_minutes: int
    campaign: ProviderHeldCampaign | Unset = UNSET
    fallback: str | Unset = UNSET
    fallback_short: str | Unset = UNSET
    primary: str | Unset = UNSET
    suggestion: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_failover_alternative import ProviderFailoverAlternative # noqa: PLC0415
        from ..models.provider_held_campaign import ProviderHeldCampaign # noqa: PLC0415
        from ..models.provider_named_count import ProviderNamedCount # noqa: PLC0415
        from ..models.vet_check import VetCheck # noqa: PLC0415
        alternatives = []
        for alternatives_item_data in self.alternatives:
            alternatives_item = alternatives_item_data.to_dict()
            alternatives.append(alternatives_item)



        fallback_checks = []
        for fallback_checks_item_data in self.fallback_checks:
            fallback_checks_item = fallback_checks_item_data.to_dict()
            fallback_checks.append(fallback_checks_item)



        policy = self.policy.value

        rerouted = self.rerouted

        rerouted_by = []
        for rerouted_by_item_data in self.rerouted_by:
            rerouted_by_item = rerouted_by_item_data.to_dict()
            rerouted_by.append(rerouted_by_item)



        role = self.role.value

        since = self.since.isoformat()

        state = self.state.value

        stream = self.stream.value

        switch_back_minutes = self.switch_back_minutes

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        fallback = self.fallback

        fallback_short = self.fallback_short

        primary = self.primary

        suggestion = self.suggestion


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alternatives": alternatives,
            "fallbackChecks": fallback_checks,
            "policy": policy,
            "rerouted": rerouted,
            "reroutedBy": rerouted_by,
            "role": role,
            "since": since,
            "state": state,
            "stream": stream,
            "switchBackMinutes": switch_back_minutes,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if fallback_short is not UNSET:
            field_dict["fallbackShort"] = fallback_short
        if primary is not UNSET:
            field_dict["primary"] = primary
        if suggestion is not UNSET:
            field_dict["suggestion"] = suggestion

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_failover_alternative import ProviderFailoverAlternative # noqa: PLC0415
        from ..models.provider_held_campaign import ProviderHeldCampaign # noqa: PLC0415
        from ..models.provider_named_count import ProviderNamedCount # noqa: PLC0415
        from ..models.vet_check import VetCheck # noqa: PLC0415
        d = dict(src_dict)
        alternatives = []
        _alternatives = d.pop("alternatives")
        for alternatives_item_data in (_alternatives):
            alternatives_item = ProviderFailoverAlternative.from_dict(alternatives_item_data)



            alternatives.append(alternatives_item)


        fallback_checks = []
        _fallback_checks = d.pop("fallbackChecks")
        for fallback_checks_item_data in (_fallback_checks):
            fallback_checks_item = VetCheck.from_dict(fallback_checks_item_data)



            fallback_checks.append(fallback_checks_item)


        policy = ProviderOutageStreamPolicy(d.pop("policy"))




        rerouted = d.pop("rerouted")

        rerouted_by = []
        _rerouted_by = d.pop("reroutedBy")
        for rerouted_by_item_data in (_rerouted_by):
            rerouted_by_item = ProviderNamedCount.from_dict(rerouted_by_item_data)



            rerouted_by.append(rerouted_by_item)


        role = ProviderOutageStreamRole(d.pop("role"))




        since = datetime.datetime.fromisoformat(d.pop("since"))




        state = ProviderOutageStreamState(d.pop("state"))




        stream = ProviderOutageStreamStream(d.pop("stream"))




        switch_back_minutes = d.pop("switchBackMinutes")

        _campaign = d.pop("campaign", UNSET)
        campaign: ProviderHeldCampaign | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = ProviderHeldCampaign.from_dict(_campaign)




        fallback = d.pop("fallback", UNSET)

        fallback_short = d.pop("fallbackShort", UNSET)

        primary = d.pop("primary", UNSET)

        suggestion = d.pop("suggestion", UNSET)

        provider_outage_stream = cls(
            alternatives=alternatives,
            fallback_checks=fallback_checks,
            policy=policy,
            rerouted=rerouted,
            rerouted_by=rerouted_by,
            role=role,
            since=since,
            state=state,
            stream=stream,
            switch_back_minutes=switch_back_minutes,
            campaign=campaign,
            fallback=fallback,
            fallback_short=fallback_short,
            primary=primary,
            suggestion=suggestion,
        )

        return provider_outage_stream

