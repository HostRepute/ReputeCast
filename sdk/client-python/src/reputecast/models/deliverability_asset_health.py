from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_asset_health_role import DeliverabilityAssetHealthRole
from ..models.deliverability_asset_health_tone import DeliverabilityAssetHealthTone
from ..models.deliverability_asset_health_type import DeliverabilityAssetHealthType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_asset_blocklists import DeliverabilityAssetBlocklists
  from ..models.deliverability_domain_auth import DeliverabilityDomainAuth
  from ..models.deliverability_reverse_dns import DeliverabilityReverseDNS
  from ..models.deliverability_snds_brief import DeliverabilitySNDSBrief
  from ..models.deliverability_warmup_progress import DeliverabilityWarmupProgress





T = TypeVar("T", bound="DeliverabilityAssetHealth")



@_attrs_define
class DeliverabilityAssetHealth:
    """ 
        Attributes:
            asset (str):
            blocklists (DeliverabilityAssetBlocklists):
            role (DeliverabilityAssetHealthRole):
            tone (DeliverabilityAssetHealthTone):
            type_ (DeliverabilityAssetHealthType):
            used_by (list[str]):
            auth (DeliverabilityDomainAuth | Unset):
            dmarc_aligned (float | Unset):
            fallback (DeliverabilityAssetBlocklists | Unset):
            not_monitored (bool | Unset):
            provider (str | Unset):
            provider_id (str | Unset):
            provider_status (str | Unset):
            reverse_dns (DeliverabilityReverseDNS | Unset):
            score (int | Unset):
            snds (DeliverabilitySNDSBrief | Unset):
            stale (datetime.datetime | Unset):
            verdict (str | Unset):
            warmup (DeliverabilityWarmupProgress | Unset):
     """

    asset: str
    blocklists: DeliverabilityAssetBlocklists
    role: DeliverabilityAssetHealthRole
    tone: DeliverabilityAssetHealthTone
    type_: DeliverabilityAssetHealthType
    used_by: list[str]
    auth: DeliverabilityDomainAuth | Unset = UNSET
    dmarc_aligned: float | Unset = UNSET
    fallback: DeliverabilityAssetBlocklists | Unset = UNSET
    not_monitored: bool | Unset = UNSET
    provider: str | Unset = UNSET
    provider_id: str | Unset = UNSET
    provider_status: str | Unset = UNSET
    reverse_dns: DeliverabilityReverseDNS | Unset = UNSET
    score: int | Unset = UNSET
    snds: DeliverabilitySNDSBrief | Unset = UNSET
    stale: datetime.datetime | Unset = UNSET
    verdict: str | Unset = UNSET
    warmup: DeliverabilityWarmupProgress | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_asset_blocklists import DeliverabilityAssetBlocklists # noqa: PLC0415
        from ..models.deliverability_domain_auth import DeliverabilityDomainAuth # noqa: PLC0415
        from ..models.deliverability_reverse_dns import DeliverabilityReverseDNS # noqa: PLC0415
        from ..models.deliverability_snds_brief import DeliverabilitySNDSBrief # noqa: PLC0415
        from ..models.deliverability_warmup_progress import DeliverabilityWarmupProgress # noqa: PLC0415
        asset = self.asset

        blocklists = self.blocklists.to_dict()

        role = self.role.value

        tone = self.tone.value

        type_ = self.type_.value

        used_by = self.used_by



        auth: dict[str, Any] | Unset = UNSET
        if not isinstance(self.auth, Unset):
            auth = self.auth.to_dict()

        dmarc_aligned = self.dmarc_aligned

        fallback: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fallback, Unset):
            fallback = self.fallback.to_dict()

        not_monitored = self.not_monitored

        provider = self.provider

        provider_id = self.provider_id

        provider_status = self.provider_status

        reverse_dns: dict[str, Any] | Unset = UNSET
        if not isinstance(self.reverse_dns, Unset):
            reverse_dns = self.reverse_dns.to_dict()

        score = self.score

        snds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.snds, Unset):
            snds = self.snds.to_dict()

        stale: str | Unset = UNSET
        if not isinstance(self.stale, Unset):
            stale = self.stale.isoformat()

        verdict = self.verdict

        warmup: dict[str, Any] | Unset = UNSET
        if not isinstance(self.warmup, Unset):
            warmup = self.warmup.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
            "blocklists": blocklists,
            "role": role,
            "tone": tone,
            "type": type_,
            "usedBy": used_by,
        })
        if auth is not UNSET:
            field_dict["auth"] = auth
        if dmarc_aligned is not UNSET:
            field_dict["dmarcAligned"] = dmarc_aligned
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if not_monitored is not UNSET:
            field_dict["notMonitored"] = not_monitored
        if provider is not UNSET:
            field_dict["provider"] = provider
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id
        if provider_status is not UNSET:
            field_dict["providerStatus"] = provider_status
        if reverse_dns is not UNSET:
            field_dict["reverseDns"] = reverse_dns
        if score is not UNSET:
            field_dict["score"] = score
        if snds is not UNSET:
            field_dict["snds"] = snds
        if stale is not UNSET:
            field_dict["stale"] = stale
        if verdict is not UNSET:
            field_dict["verdict"] = verdict
        if warmup is not UNSET:
            field_dict["warmup"] = warmup

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_asset_blocklists import DeliverabilityAssetBlocklists # noqa: PLC0415
        from ..models.deliverability_domain_auth import DeliverabilityDomainAuth # noqa: PLC0415
        from ..models.deliverability_reverse_dns import DeliverabilityReverseDNS # noqa: PLC0415
        from ..models.deliverability_snds_brief import DeliverabilitySNDSBrief # noqa: PLC0415
        from ..models.deliverability_warmup_progress import DeliverabilityWarmupProgress # noqa: PLC0415
        d = dict(src_dict)
        asset = d.pop("asset")

        blocklists = DeliverabilityAssetBlocklists.from_dict(d.pop("blocklists"))




        role = DeliverabilityAssetHealthRole(d.pop("role"))




        tone = DeliverabilityAssetHealthTone(d.pop("tone"))




        type_ = DeliverabilityAssetHealthType(d.pop("type"))




        used_by = cast(list[str], d.pop("usedBy"))


        _auth = d.pop("auth", UNSET)
        auth: DeliverabilityDomainAuth | Unset
        if isinstance(_auth,  Unset):
            auth = UNSET
        else:
            auth = DeliverabilityDomainAuth.from_dict(_auth)




        dmarc_aligned = d.pop("dmarcAligned", UNSET)

        _fallback = d.pop("fallback", UNSET)
        fallback: DeliverabilityAssetBlocklists | Unset
        if isinstance(_fallback,  Unset):
            fallback = UNSET
        else:
            fallback = DeliverabilityAssetBlocklists.from_dict(_fallback)




        not_monitored = d.pop("notMonitored", UNSET)

        provider = d.pop("provider", UNSET)

        provider_id = d.pop("providerId", UNSET)

        provider_status = d.pop("providerStatus", UNSET)

        _reverse_dns = d.pop("reverseDns", UNSET)
        reverse_dns: DeliverabilityReverseDNS | Unset
        if isinstance(_reverse_dns,  Unset):
            reverse_dns = UNSET
        else:
            reverse_dns = DeliverabilityReverseDNS.from_dict(_reverse_dns)




        score = d.pop("score", UNSET)

        _snds = d.pop("snds", UNSET)
        snds: DeliverabilitySNDSBrief | Unset
        if isinstance(_snds,  Unset):
            snds = UNSET
        else:
            snds = DeliverabilitySNDSBrief.from_dict(_snds)




        _stale = d.pop("stale", UNSET)
        stale: datetime.datetime | Unset
        if isinstance(_stale,  Unset):
            stale = UNSET
        else:
            stale = datetime.datetime.fromisoformat(_stale)




        verdict = d.pop("verdict", UNSET)

        _warmup = d.pop("warmup", UNSET)
        warmup: DeliverabilityWarmupProgress | Unset
        if isinstance(_warmup,  Unset):
            warmup = UNSET
        else:
            warmup = DeliverabilityWarmupProgress.from_dict(_warmup)




        deliverability_asset_health = cls(
            asset=asset,
            blocklists=blocklists,
            role=role,
            tone=tone,
            type_=type_,
            used_by=used_by,
            auth=auth,
            dmarc_aligned=dmarc_aligned,
            fallback=fallback,
            not_monitored=not_monitored,
            provider=provider,
            provider_id=provider_id,
            provider_status=provider_status,
            reverse_dns=reverse_dns,
            score=score,
            snds=snds,
            stale=stale,
            verdict=verdict,
            warmup=warmup,
        )

        return deliverability_asset_health

