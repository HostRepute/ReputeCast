from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_sender_health_label import OverviewSenderHealthLabel
from ..models.overview_sender_health_state import OverviewSenderHealthState
from ..models.overview_sender_health_tone import OverviewSenderHealthTone
from ..models.overview_sender_health_via import OverviewSenderHealthVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.overview_health_reason import OverviewHealthReason





T = TypeVar("T", bound="OverviewSenderHealth")



@_attrs_define
class OverviewSenderHealth:
    """ 
        Attributes:
            domain (str):
            label (OverviewSenderHealthLabel):
            reasons (list[OverviewHealthReason]):
            score (int):
            sources (int): Blocklists checked
            state (OverviewSenderHealthState):
            tone (OverviewSenderHealthTone):
            via (OverviewSenderHealthVia):
            checked_at (datetime.datetime | Unset):
     """

    domain: str
    label: OverviewSenderHealthLabel
    reasons: list[OverviewHealthReason]
    score: int
    sources: int
    state: OverviewSenderHealthState
    tone: OverviewSenderHealthTone
    via: OverviewSenderHealthVia
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_health_reason import OverviewHealthReason # noqa: PLC0415
        domain = self.domain

        label = self.label.value

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        score = self.score

        sources = self.sources

        state = self.state.value

        tone = self.tone.value

        via = self.via.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "label": label,
            "reasons": reasons,
            "score": score,
            "sources": sources,
            "state": state,
            "tone": tone,
            "via": via,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_health_reason import OverviewHealthReason # noqa: PLC0415
        d = dict(src_dict)
        domain = d.pop("domain")

        label = OverviewSenderHealthLabel(d.pop("label"))




        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = OverviewHealthReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        score = d.pop("score")

        sources = d.pop("sources")

        state = OverviewSenderHealthState(d.pop("state"))




        tone = OverviewSenderHealthTone(d.pop("tone"))




        via = OverviewSenderHealthVia(d.pop("via"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        overview_sender_health = cls(
            domain=domain,
            label=label,
            reasons=reasons,
            score=score,
            sources=sources,
            state=state,
            tone=tone,
            via=via,
            checked_at=checked_at,
        )

        return overview_sender_health

