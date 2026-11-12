from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_fix_spec_kind import AccessibilityFixSpecKind
from ..models.accessibility_fix_spec_target import AccessibilityFixSpecTarget
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.accessibility_color_suggestion import AccessibilityColorSuggestion
  from ..models.accessibility_heading_change import AccessibilityHeadingChange
  from ..models.accessibility_link_target import AccessibilityLinkTarget





T = TypeVar("T", bound="AccessibilityFixSpec")



@_attrs_define
class AccessibilityFixSpec:
    """ 
        Attributes:
            decorative (bool):
            headings (list[AccessibilityHeadingChange]):
            kind (AccessibilityFixSpecKind):
            links (list[AccessibilityLinkTarget]):
            one_click (bool):
            per_language (bool):
            suggestions (list[AccessibilityColorSuggestion]):
            against (str | Unset):
            from_ (str | Unset):
            from_px (float | Unset):
            large (bool | Unset):
            needs (float | Unset):
            ratio (float | Unset):
            subject (str | Unset):
            target (AccessibilityFixSpecTarget | Unset):
            to_px (int | Unset):
     """

    decorative: bool
    headings: list[AccessibilityHeadingChange]
    kind: AccessibilityFixSpecKind
    links: list[AccessibilityLinkTarget]
    one_click: bool
    per_language: bool
    suggestions: list[AccessibilityColorSuggestion]
    against: str | Unset = UNSET
    from_: str | Unset = UNSET
    from_px: float | Unset = UNSET
    large: bool | Unset = UNSET
    needs: float | Unset = UNSET
    ratio: float | Unset = UNSET
    subject: str | Unset = UNSET
    target: AccessibilityFixSpecTarget | Unset = UNSET
    to_px: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_color_suggestion import AccessibilityColorSuggestion # noqa: PLC0415
        from ..models.accessibility_heading_change import AccessibilityHeadingChange # noqa: PLC0415
        from ..models.accessibility_link_target import AccessibilityLinkTarget # noqa: PLC0415
        decorative = self.decorative

        headings = []
        for headings_item_data in self.headings:
            headings_item = headings_item_data.to_dict()
            headings.append(headings_item)



        kind = self.kind.value

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        one_click = self.one_click

        per_language = self.per_language

        suggestions = []
        for suggestions_item_data in self.suggestions:
            suggestions_item = suggestions_item_data.to_dict()
            suggestions.append(suggestions_item)



        against = self.against

        from_ = self.from_

        from_px = self.from_px

        large = self.large

        needs = self.needs

        ratio = self.ratio

        subject = self.subject

        target: str | Unset = UNSET
        if not isinstance(self.target, Unset):
            target = self.target.value


        to_px = self.to_px


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "decorative": decorative,
            "headings": headings,
            "kind": kind,
            "links": links,
            "oneClick": one_click,
            "perLanguage": per_language,
            "suggestions": suggestions,
        })
        if against is not UNSET:
            field_dict["against"] = against
        if from_ is not UNSET:
            field_dict["from"] = from_
        if from_px is not UNSET:
            field_dict["fromPx"] = from_px
        if large is not UNSET:
            field_dict["large"] = large
        if needs is not UNSET:
            field_dict["needs"] = needs
        if ratio is not UNSET:
            field_dict["ratio"] = ratio
        if subject is not UNSET:
            field_dict["subject"] = subject
        if target is not UNSET:
            field_dict["target"] = target
        if to_px is not UNSET:
            field_dict["toPx"] = to_px

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_color_suggestion import AccessibilityColorSuggestion # noqa: PLC0415
        from ..models.accessibility_heading_change import AccessibilityHeadingChange # noqa: PLC0415
        from ..models.accessibility_link_target import AccessibilityLinkTarget # noqa: PLC0415
        d = dict(src_dict)
        decorative = d.pop("decorative")

        headings = []
        _headings = d.pop("headings")
        for headings_item_data in (_headings):
            headings_item = AccessibilityHeadingChange.from_dict(headings_item_data)



            headings.append(headings_item)


        kind = AccessibilityFixSpecKind(d.pop("kind"))




        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = AccessibilityLinkTarget.from_dict(links_item_data)



            links.append(links_item)


        one_click = d.pop("oneClick")

        per_language = d.pop("perLanguage")

        suggestions = []
        _suggestions = d.pop("suggestions")
        for suggestions_item_data in (_suggestions):
            suggestions_item = AccessibilityColorSuggestion.from_dict(suggestions_item_data)



            suggestions.append(suggestions_item)


        against = d.pop("against", UNSET)

        from_ = d.pop("from", UNSET)

        from_px = d.pop("fromPx", UNSET)

        large = d.pop("large", UNSET)

        needs = d.pop("needs", UNSET)

        ratio = d.pop("ratio", UNSET)

        subject = d.pop("subject", UNSET)

        _target = d.pop("target", UNSET)
        target: AccessibilityFixSpecTarget | Unset
        if isinstance(_target,  Unset):
            target = UNSET
        else:
            target = AccessibilityFixSpecTarget(_target)




        to_px = d.pop("toPx", UNSET)

        accessibility_fix_spec = cls(
            decorative=decorative,
            headings=headings,
            kind=kind,
            links=links,
            one_click=one_click,
            per_language=per_language,
            suggestions=suggestions,
            against=against,
            from_=from_,
            from_px=from_px,
            large=large,
            needs=needs,
            ratio=ratio,
            subject=subject,
            target=target,
            to_px=to_px,
        )

        return accessibility_fix_spec

