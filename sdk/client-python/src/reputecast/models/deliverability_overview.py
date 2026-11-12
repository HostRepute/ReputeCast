from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.deliverability_asset_health import DeliverabilityAssetHealth
  from ..models.deliverability_blocklist_status import DeliverabilityBlocklistStatus
  from ..models.deliverability_checks import DeliverabilityChecks
  from ..models.deliverability_dmarc_summary import DeliverabilityDMARCSummary
  from ..models.deliverability_host_repute import DeliverabilityHostRepute
  from ..models.deliverability_incident_timeline import DeliverabilityIncidentTimeline
  from ..models.deliverability_pool_health import DeliverabilityPoolHealth
  from ..models.deliverability_postmaster import DeliverabilityPostmaster
  from ..models.deliverability_provider_health import DeliverabilityProviderHealth
  from ..models.deliverability_snds import DeliverabilitySNDS





T = TypeVar("T", bound="DeliverabilityOverview")



@_attrs_define
class DeliverabilityOverview:
    """ 
        Attributes:
            assets (list[DeliverabilityAssetHealth]):
            blocklists (DeliverabilityBlocklistStatus):
            checks (DeliverabilityChecks):
            dmarc (DeliverabilityDMARCSummary):
            host_repute (DeliverabilityHostRepute):
            incidents (DeliverabilityIncidentTimeline):
            pools (list[DeliverabilityPoolHealth]):
            providers (list[DeliverabilityProviderHealth]):
            postmaster (DeliverabilityPostmaster | Unset):
            snds (DeliverabilitySNDS | Unset):
     """

    assets: list[DeliverabilityAssetHealth]
    blocklists: DeliverabilityBlocklistStatus
    checks: DeliverabilityChecks
    dmarc: DeliverabilityDMARCSummary
    host_repute: DeliverabilityHostRepute
    incidents: DeliverabilityIncidentTimeline
    pools: list[DeliverabilityPoolHealth]
    providers: list[DeliverabilityProviderHealth]
    postmaster: DeliverabilityPostmaster | Unset = UNSET
    snds: DeliverabilitySNDS | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_asset_health import DeliverabilityAssetHealth # noqa: PLC0415
        from ..models.deliverability_blocklist_status import DeliverabilityBlocklistStatus # noqa: PLC0415
        from ..models.deliverability_checks import DeliverabilityChecks # noqa: PLC0415
        from ..models.deliverability_dmarc_summary import DeliverabilityDMARCSummary # noqa: PLC0415
        from ..models.deliverability_host_repute import DeliverabilityHostRepute # noqa: PLC0415
        from ..models.deliverability_incident_timeline import DeliverabilityIncidentTimeline # noqa: PLC0415
        from ..models.deliverability_pool_health import DeliverabilityPoolHealth # noqa: PLC0415
        from ..models.deliverability_postmaster import DeliverabilityPostmaster # noqa: PLC0415
        from ..models.deliverability_provider_health import DeliverabilityProviderHealth # noqa: PLC0415
        from ..models.deliverability_snds import DeliverabilitySNDS # noqa: PLC0415
        assets = []
        for assets_item_data in self.assets:
            assets_item = assets_item_data.to_dict()
            assets.append(assets_item)



        blocklists = self.blocklists.to_dict()

        checks = self.checks.to_dict()

        dmarc = self.dmarc.to_dict()

        host_repute = self.host_repute.to_dict()

        incidents = self.incidents.to_dict()

        pools = []
        for pools_item_data in self.pools:
            pools_item = pools_item_data.to_dict()
            pools.append(pools_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        postmaster: dict[str, Any] | Unset = UNSET
        if not isinstance(self.postmaster, Unset):
            postmaster = self.postmaster.to_dict()

        snds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.snds, Unset):
            snds = self.snds.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "assets": assets,
            "blocklists": blocklists,
            "checks": checks,
            "dmarc": dmarc,
            "hostRepute": host_repute,
            "incidents": incidents,
            "pools": pools,
            "providers": providers,
        })
        if postmaster is not UNSET:
            field_dict["postmaster"] = postmaster
        if snds is not UNSET:
            field_dict["snds"] = snds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_asset_health import DeliverabilityAssetHealth # noqa: PLC0415
        from ..models.deliverability_blocklist_status import DeliverabilityBlocklistStatus # noqa: PLC0415
        from ..models.deliverability_checks import DeliverabilityChecks # noqa: PLC0415
        from ..models.deliverability_dmarc_summary import DeliverabilityDMARCSummary # noqa: PLC0415
        from ..models.deliverability_host_repute import DeliverabilityHostRepute # noqa: PLC0415
        from ..models.deliverability_incident_timeline import DeliverabilityIncidentTimeline # noqa: PLC0415
        from ..models.deliverability_pool_health import DeliverabilityPoolHealth # noqa: PLC0415
        from ..models.deliverability_postmaster import DeliverabilityPostmaster # noqa: PLC0415
        from ..models.deliverability_provider_health import DeliverabilityProviderHealth # noqa: PLC0415
        from ..models.deliverability_snds import DeliverabilitySNDS # noqa: PLC0415
        d = dict(src_dict)
        assets = []
        _assets = d.pop("assets")
        for assets_item_data in (_assets):
            assets_item = DeliverabilityAssetHealth.from_dict(assets_item_data)



            assets.append(assets_item)


        blocklists = DeliverabilityBlocklistStatus.from_dict(d.pop("blocklists"))




        checks = DeliverabilityChecks.from_dict(d.pop("checks"))




        dmarc = DeliverabilityDMARCSummary.from_dict(d.pop("dmarc"))




        host_repute = DeliverabilityHostRepute.from_dict(d.pop("hostRepute"))




        incidents = DeliverabilityIncidentTimeline.from_dict(d.pop("incidents"))




        pools = []
        _pools = d.pop("pools")
        for pools_item_data in (_pools):
            pools_item = DeliverabilityPoolHealth.from_dict(pools_item_data)



            pools.append(pools_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = DeliverabilityProviderHealth.from_dict(providers_item_data)



            providers.append(providers_item)


        _postmaster = d.pop("postmaster", UNSET)
        postmaster: DeliverabilityPostmaster | Unset
        if isinstance(_postmaster,  Unset):
            postmaster = UNSET
        else:
            postmaster = DeliverabilityPostmaster.from_dict(_postmaster)




        _snds = d.pop("snds", UNSET)
        snds: DeliverabilitySNDS | Unset
        if isinstance(_snds,  Unset):
            snds = UNSET
        else:
            snds = DeliverabilitySNDS.from_dict(_snds)




        deliverability_overview = cls(
            assets=assets,
            blocklists=blocklists,
            checks=checks,
            dmarc=dmarc,
            host_repute=host_repute,
            incidents=incidents,
            pools=pools,
            providers=providers,
            postmaster=postmaster,
            snds=snds,
        )

        return deliverability_overview

