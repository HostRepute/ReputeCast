from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.scoring_band_counts import ScoringBandCounts
  from ..models.scoring_bucket import ScoringBucket
  from ..models.scoring_config import ScoringConfig
  from ..models.scoring_options import ScoringOptions
  from ..models.scoring_recalc_info import ScoringRecalcInfo
  from ..models.scoring_rule import ScoringRule
  from ..models.scoring_tier_example import ScoringTierExample
  from ..models.scoring_tier_row import ScoringTierRow
  from ..models.scoring_use import ScoringUse





T = TypeVar("T", bound="ScoringPage")



@_attrs_define
class ScoringPage:
    """ 
        Attributes:
            applied_version (int):
            bands (ScoringBandCounts):
            by_tier (list[ScoringTierRow]):
            config (ScoringConfig):
            configured (bool):
            contacts (int):
            helpers (list[str]):
            options (ScoringOptions):
            set_up_by (str):
            spread (list[ScoringBucket]):
            suggested (list[ScoringRule]):
            used_by (list[ScoringUse]):
            version (int):
            recalculation (ScoringRecalcInfo | Unset):
            set_up_at (datetime.datetime | Unset):
            tier_example (ScoringTierExample | Unset):
     """

    applied_version: int
    bands: ScoringBandCounts
    by_tier: list[ScoringTierRow]
    config: ScoringConfig
    configured: bool
    contacts: int
    helpers: list[str]
    options: ScoringOptions
    set_up_by: str
    spread: list[ScoringBucket]
    suggested: list[ScoringRule]
    used_by: list[ScoringUse]
    version: int
    recalculation: ScoringRecalcInfo | Unset = UNSET
    set_up_at: datetime.datetime | Unset = UNSET
    tier_example: ScoringTierExample | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_bucket import ScoringBucket # noqa: PLC0415
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        from ..models.scoring_options import ScoringOptions # noqa: PLC0415
        from ..models.scoring_recalc_info import ScoringRecalcInfo # noqa: PLC0415
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        from ..models.scoring_tier_example import ScoringTierExample # noqa: PLC0415
        from ..models.scoring_tier_row import ScoringTierRow # noqa: PLC0415
        from ..models.scoring_use import ScoringUse # noqa: PLC0415
        applied_version = self.applied_version

        bands = self.bands.to_dict()

        by_tier = []
        for by_tier_item_data in self.by_tier:
            by_tier_item = by_tier_item_data.to_dict()
            by_tier.append(by_tier_item)



        config = self.config.to_dict()

        configured = self.configured

        contacts = self.contacts

        helpers = self.helpers



        options = self.options.to_dict()

        set_up_by = self.set_up_by

        spread = []
        for spread_item_data in self.spread:
            spread_item = spread_item_data.to_dict()
            spread.append(spread_item)



        suggested = []
        for suggested_item_data in self.suggested:
            suggested_item = suggested_item_data.to_dict()
            suggested.append(suggested_item)



        used_by = []
        for used_by_item_data in self.used_by:
            used_by_item = used_by_item_data.to_dict()
            used_by.append(used_by_item)



        version = self.version

        recalculation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.recalculation, Unset):
            recalculation = self.recalculation.to_dict()

        set_up_at: str | Unset = UNSET
        if not isinstance(self.set_up_at, Unset):
            set_up_at = self.set_up_at.isoformat()

        tier_example: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tier_example, Unset):
            tier_example = self.tier_example.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "appliedVersion": applied_version,
            "bands": bands,
            "byTier": by_tier,
            "config": config,
            "configured": configured,
            "contacts": contacts,
            "helpers": helpers,
            "options": options,
            "setUpBy": set_up_by,
            "spread": spread,
            "suggested": suggested,
            "usedBy": used_by,
            "version": version,
        })
        if recalculation is not UNSET:
            field_dict["recalculation"] = recalculation
        if set_up_at is not UNSET:
            field_dict["setUpAt"] = set_up_at
        if tier_example is not UNSET:
            field_dict["tierExample"] = tier_example

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_band_counts import ScoringBandCounts # noqa: PLC0415
        from ..models.scoring_bucket import ScoringBucket # noqa: PLC0415
        from ..models.scoring_config import ScoringConfig # noqa: PLC0415
        from ..models.scoring_options import ScoringOptions # noqa: PLC0415
        from ..models.scoring_recalc_info import ScoringRecalcInfo # noqa: PLC0415
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        from ..models.scoring_tier_example import ScoringTierExample # noqa: PLC0415
        from ..models.scoring_tier_row import ScoringTierRow # noqa: PLC0415
        from ..models.scoring_use import ScoringUse # noqa: PLC0415
        d = dict(src_dict)
        applied_version = d.pop("appliedVersion")

        bands = ScoringBandCounts.from_dict(d.pop("bands"))




        by_tier = []
        _by_tier = d.pop("byTier")
        for by_tier_item_data in (_by_tier):
            by_tier_item = ScoringTierRow.from_dict(by_tier_item_data)



            by_tier.append(by_tier_item)


        config = ScoringConfig.from_dict(d.pop("config"))




        configured = d.pop("configured")

        contacts = d.pop("contacts")

        helpers = cast(list[str], d.pop("helpers"))


        options = ScoringOptions.from_dict(d.pop("options"))




        set_up_by = d.pop("setUpBy")

        spread = []
        _spread = d.pop("spread")
        for spread_item_data in (_spread):
            spread_item = ScoringBucket.from_dict(spread_item_data)



            spread.append(spread_item)


        suggested = []
        _suggested = d.pop("suggested")
        for suggested_item_data in (_suggested):
            suggested_item = ScoringRule.from_dict(suggested_item_data)



            suggested.append(suggested_item)


        used_by = []
        _used_by = d.pop("usedBy")
        for used_by_item_data in (_used_by):
            used_by_item = ScoringUse.from_dict(used_by_item_data)



            used_by.append(used_by_item)


        version = d.pop("version")

        _recalculation = d.pop("recalculation", UNSET)
        recalculation: ScoringRecalcInfo | Unset
        if isinstance(_recalculation,  Unset):
            recalculation = UNSET
        else:
            recalculation = ScoringRecalcInfo.from_dict(_recalculation)




        _set_up_at = d.pop("setUpAt", UNSET)
        set_up_at: datetime.datetime | Unset
        if isinstance(_set_up_at,  Unset):
            set_up_at = UNSET
        else:
            set_up_at = datetime.datetime.fromisoformat(_set_up_at)




        _tier_example = d.pop("tierExample", UNSET)
        tier_example: ScoringTierExample | Unset
        if isinstance(_tier_example,  Unset):
            tier_example = UNSET
        else:
            tier_example = ScoringTierExample.from_dict(_tier_example)




        scoring_page = cls(
            applied_version=applied_version,
            bands=bands,
            by_tier=by_tier,
            config=config,
            configured=configured,
            contacts=contacts,
            helpers=helpers,
            options=options,
            set_up_by=set_up_by,
            spread=spread,
            suggested=suggested,
            used_by=used_by,
            version=version,
            recalculation=recalculation,
            set_up_at=set_up_at,
            tier_example=tier_example,
        )

        return scoring_page

