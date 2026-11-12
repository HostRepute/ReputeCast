from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_insight_action import ReportInsightAction
from ..models.report_insight_kind import ReportInsightKind
from ..models.report_insight_tone import ReportInsightTone
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportInsight")



@_attrs_define
class ReportInsight:
    """ 
        Attributes:
            detail (str):
            kind (ReportInsightKind):
            title (str):
            tone (ReportInsightTone):
            action (ReportInsightAction | Unset):
            action_label (str | Unset):
            link (str | Unset):
            position (int | Unset):
            variant (str | Unset):
     """

    detail: str
    kind: ReportInsightKind
    title: str
    tone: ReportInsightTone
    action: ReportInsightAction | Unset = UNSET
    action_label: str | Unset = UNSET
    link: str | Unset = UNSET
    position: int | Unset = UNSET
    variant: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        kind = self.kind.value

        title = self.title

        tone = self.tone.value

        action: str | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.value


        action_label = self.action_label

        link = self.link

        position = self.position

        variant = self.variant


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "kind": kind,
            "title": title,
            "tone": tone,
        })
        if action is not UNSET:
            field_dict["action"] = action
        if action_label is not UNSET:
            field_dict["actionLabel"] = action_label
        if link is not UNSET:
            field_dict["link"] = link
        if position is not UNSET:
            field_dict["position"] = position
        if variant is not UNSET:
            field_dict["variant"] = variant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        kind = ReportInsightKind(d.pop("kind"))




        title = d.pop("title")

        tone = ReportInsightTone(d.pop("tone"))




        _action = d.pop("action", UNSET)
        action: ReportInsightAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = ReportInsightAction(_action)




        action_label = d.pop("actionLabel", UNSET)

        link = d.pop("link", UNSET)

        position = d.pop("position", UNSET)

        variant = d.pop("variant", UNSET)

        report_insight = cls(
            detail=detail,
            kind=kind,
            title=title,
            tone=tone,
            action=action,
            action_label=action_label,
            link=link,
            position=position,
            variant=variant,
        )

        return report_insight

