from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.provider_auth_failure import ProviderAuthFailure
  from ..models.provider_card import ProviderCard
  from ..models.provider_outage import ProviderOutage
  from ..models.provider_stream_route import ProviderStreamRoute
  from ..models.provider_webhooks_silent import ProviderWebhooksSilent
  from ..models.providers_summary import ProvidersSummary





T = TypeVar("T", bound="ProvidersOverview")



@_attrs_define
class ProvidersOverview:
    """ 
        Attributes:
            providers (list[ProviderCard]):
            routing (list[ProviderStreamRoute]):
            summary (ProvidersSummary):
            auth_failure (ProviderAuthFailure | Unset):
            outage (ProviderOutage | Unset):
            silent (ProviderWebhooksSilent | Unset):
     """

    providers: list[ProviderCard]
    routing: list[ProviderStreamRoute]
    summary: ProvidersSummary
    auth_failure: ProviderAuthFailure | Unset = UNSET
    outage: ProviderOutage | Unset = UNSET
    silent: ProviderWebhooksSilent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_auth_failure import ProviderAuthFailure # noqa: PLC0415
        from ..models.provider_card import ProviderCard # noqa: PLC0415
        from ..models.provider_outage import ProviderOutage # noqa: PLC0415
        from ..models.provider_stream_route import ProviderStreamRoute # noqa: PLC0415
        from ..models.provider_webhooks_silent import ProviderWebhooksSilent # noqa: PLC0415
        from ..models.providers_summary import ProvidersSummary # noqa: PLC0415
        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        routing = []
        for routing_item_data in self.routing:
            routing_item = routing_item_data.to_dict()
            routing.append(routing_item)



        summary = self.summary.to_dict()

        auth_failure: dict[str, Any] | Unset = UNSET
        if not isinstance(self.auth_failure, Unset):
            auth_failure = self.auth_failure.to_dict()

        outage: dict[str, Any] | Unset = UNSET
        if not isinstance(self.outage, Unset):
            outage = self.outage.to_dict()

        silent: dict[str, Any] | Unset = UNSET
        if not isinstance(self.silent, Unset):
            silent = self.silent.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "providers": providers,
            "routing": routing,
            "summary": summary,
        })
        if auth_failure is not UNSET:
            field_dict["authFailure"] = auth_failure
        if outage is not UNSET:
            field_dict["outage"] = outage
        if silent is not UNSET:
            field_dict["silent"] = silent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_auth_failure import ProviderAuthFailure # noqa: PLC0415
        from ..models.provider_card import ProviderCard # noqa: PLC0415
        from ..models.provider_outage import ProviderOutage # noqa: PLC0415
        from ..models.provider_stream_route import ProviderStreamRoute # noqa: PLC0415
        from ..models.provider_webhooks_silent import ProviderWebhooksSilent # noqa: PLC0415
        from ..models.providers_summary import ProvidersSummary # noqa: PLC0415
        d = dict(src_dict)
        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = ProviderCard.from_dict(providers_item_data)



            providers.append(providers_item)


        routing = []
        _routing = d.pop("routing")
        for routing_item_data in (_routing):
            routing_item = ProviderStreamRoute.from_dict(routing_item_data)



            routing.append(routing_item)


        summary = ProvidersSummary.from_dict(d.pop("summary"))




        _auth_failure = d.pop("authFailure", UNSET)
        auth_failure: ProviderAuthFailure | Unset
        if isinstance(_auth_failure,  Unset):
            auth_failure = UNSET
        else:
            auth_failure = ProviderAuthFailure.from_dict(_auth_failure)




        _outage = d.pop("outage", UNSET)
        outage: ProviderOutage | Unset
        if isinstance(_outage,  Unset):
            outage = UNSET
        else:
            outage = ProviderOutage.from_dict(_outage)




        _silent = d.pop("silent", UNSET)
        silent: ProviderWebhooksSilent | Unset
        if isinstance(_silent,  Unset):
            silent = UNSET
        else:
            silent = ProviderWebhooksSilent.from_dict(_silent)




        providers_overview = cls(
            providers=providers,
            routing=routing,
            summary=summary,
            auth_failure=auth_failure,
            outage=outage,
            silent=silent,
        )

        return providers_overview

