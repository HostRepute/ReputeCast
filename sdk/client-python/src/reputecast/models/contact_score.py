from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_score_band import ContactScoreBand
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_score_change import ContactScoreChange
  from ..models.contact_score_decay import ContactScoreDecay
  from ..models.contact_score_point import ContactScorePoint
  from ..models.contact_score_rule import ContactScoreRule
  from ..models.contact_score_use import ContactScoreUse
  from ..models.scoring_bands import ScoringBands





T = TypeVar("T", bound="ContactScore")



@_attrs_define
class ContactScore:
    """ 
        Attributes:
            band (ContactScoreBand):
            bands (ScoringBands):
            breakdown (list[ContactScoreRule]):
            configured (bool):
            decay (ContactScoreDecay):
            joined_at (datetime.datetime):
            line (list[ContactScorePoint]):
            recalculating (bool):
            score (int):
            uses (list[ContactScoreUse]):
            band_since (datetime.datetime | Unset):
            last_change (ContactScoreChange | Unset):
     """

    band: ContactScoreBand
    bands: ScoringBands
    breakdown: list[ContactScoreRule]
    configured: bool
    decay: ContactScoreDecay
    joined_at: datetime.datetime
    line: list[ContactScorePoint]
    recalculating: bool
    score: int
    uses: list[ContactScoreUse]
    band_since: datetime.datetime | Unset = UNSET
    last_change: ContactScoreChange | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_score_change import ContactScoreChange # noqa: PLC0415
        from ..models.contact_score_decay import ContactScoreDecay # noqa: PLC0415
        from ..models.contact_score_point import ContactScorePoint # noqa: PLC0415
        from ..models.contact_score_rule import ContactScoreRule # noqa: PLC0415
        from ..models.contact_score_use import ContactScoreUse # noqa: PLC0415
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        band = self.band.value

        bands = self.bands.to_dict()

        breakdown = []
        for breakdown_item_data in self.breakdown:
            breakdown_item = breakdown_item_data.to_dict()
            breakdown.append(breakdown_item)



        configured = self.configured

        decay = self.decay.to_dict()

        joined_at = self.joined_at.isoformat()

        line = []
        for line_item_data in self.line:
            line_item = line_item_data.to_dict()
            line.append(line_item)



        recalculating = self.recalculating

        score = self.score

        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)



        band_since: str | Unset = UNSET
        if not isinstance(self.band_since, Unset):
            band_since = self.band_since.isoformat()

        last_change: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_change, Unset):
            last_change = self.last_change.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "band": band,
            "bands": bands,
            "breakdown": breakdown,
            "configured": configured,
            "decay": decay,
            "joinedAt": joined_at,
            "line": line,
            "recalculating": recalculating,
            "score": score,
            "uses": uses,
        })
        if band_since is not UNSET:
            field_dict["bandSince"] = band_since
        if last_change is not UNSET:
            field_dict["lastChange"] = last_change

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_score_change import ContactScoreChange # noqa: PLC0415
        from ..models.contact_score_decay import ContactScoreDecay # noqa: PLC0415
        from ..models.contact_score_point import ContactScorePoint # noqa: PLC0415
        from ..models.contact_score_rule import ContactScoreRule # noqa: PLC0415
        from ..models.contact_score_use import ContactScoreUse # noqa: PLC0415
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        d = dict(src_dict)
        band = ContactScoreBand(d.pop("band"))




        bands = ScoringBands.from_dict(d.pop("bands"))




        breakdown = []
        _breakdown = d.pop("breakdown")
        for breakdown_item_data in (_breakdown):
            breakdown_item = ContactScoreRule.from_dict(breakdown_item_data)



            breakdown.append(breakdown_item)


        configured = d.pop("configured")

        decay = ContactScoreDecay.from_dict(d.pop("decay"))




        joined_at = datetime.datetime.fromisoformat(d.pop("joinedAt"))




        line = []
        _line = d.pop("line")
        for line_item_data in (_line):
            line_item = ContactScorePoint.from_dict(line_item_data)



            line.append(line_item)


        recalculating = d.pop("recalculating")

        score = d.pop("score")

        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = ContactScoreUse.from_dict(uses_item_data)



            uses.append(uses_item)


        _band_since = d.pop("bandSince", UNSET)
        band_since: datetime.datetime | Unset
        if isinstance(_band_since,  Unset):
            band_since = UNSET
        else:
            band_since = datetime.datetime.fromisoformat(_band_since)




        _last_change = d.pop("lastChange", UNSET)
        last_change: ContactScoreChange | Unset
        if isinstance(_last_change,  Unset):
            last_change = UNSET
        else:
            last_change = ContactScoreChange.from_dict(_last_change)




        contact_score = cls(
            band=band,
            bands=bands,
            breakdown=breakdown,
            configured=configured,
            decay=decay,
            joined_at=joined_at,
            line=line,
            recalculating=recalculating,
            score=score,
            uses=uses,
            band_since=band_since,
            last_change=last_change,
        )

        return contact_score

