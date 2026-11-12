from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_fix_input_kind import AccessibilityFixInputKind
from ..models.accessibility_fix_input_role import AccessibilityFixInputRole
from ..models.accessibility_fix_input_target import AccessibilityFixInputTarget
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.accessibility_heading_change import AccessibilityHeadingChange





T = TypeVar("T", bound="AccessibilityFixInput")



@_attrs_define
class AccessibilityFixInput:
    """ 
        Attributes:
            kind (AccessibilityFixInputKind):
            mjml (str):
            path (list[int]):
            from_ (str | Unset):
            headings (list[AccessibilityHeadingChange] | Unset):
            index (int | Unset):
            role (AccessibilityFixInputRole | Unset):
            target (AccessibilityFixInputTarget | Unset):
            to_px (int | Unset):
            value (str | Unset):
     """

    kind: AccessibilityFixInputKind
    mjml: str
    path: list[int]
    from_: str | Unset = UNSET
    headings: list[AccessibilityHeadingChange] | Unset = UNSET
    index: int | Unset = UNSET
    role: AccessibilityFixInputRole | Unset = UNSET
    target: AccessibilityFixInputTarget | Unset = UNSET
    to_px: int | Unset = UNSET
    value: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_heading_change import AccessibilityHeadingChange # noqa: PLC0415
        kind = self.kind.value

        mjml = self.mjml

        path = self.path



        from_ = self.from_

        headings: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.headings, Unset):
            headings = []
            for headings_item_data in self.headings:
                headings_item = headings_item_data.to_dict()
                headings.append(headings_item)



        index = self.index

        role: str | Unset = UNSET
        if not isinstance(self.role, Unset):
            role = self.role.value


        target: str | Unset = UNSET
        if not isinstance(self.target, Unset):
            target = self.target.value


        to_px = self.to_px

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "mjml": mjml,
            "path": path,
        })
        if from_ is not UNSET:
            field_dict["from"] = from_
        if headings is not UNSET:
            field_dict["headings"] = headings
        if index is not UNSET:
            field_dict["index"] = index
        if role is not UNSET:
            field_dict["role"] = role
        if target is not UNSET:
            field_dict["target"] = target
        if to_px is not UNSET:
            field_dict["toPx"] = to_px
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_heading_change import AccessibilityHeadingChange # noqa: PLC0415
        d = dict(src_dict)
        kind = AccessibilityFixInputKind(d.pop("kind"))




        mjml = d.pop("mjml")

        path = cast(list[int], d.pop("path"))


        from_ = d.pop("from", UNSET)

        _headings = d.pop("headings", UNSET)
        headings: list[AccessibilityHeadingChange] | Unset = UNSET
        if _headings is not UNSET:
            headings = []
            for headings_item_data in _headings:
                headings_item = AccessibilityHeadingChange.from_dict(headings_item_data)



                headings.append(headings_item)


        index = d.pop("index", UNSET)

        _role = d.pop("role", UNSET)
        role: AccessibilityFixInputRole | Unset
        if isinstance(_role,  Unset):
            role = UNSET
        else:
            role = AccessibilityFixInputRole(_role)




        _target = d.pop("target", UNSET)
        target: AccessibilityFixInputTarget | Unset
        if isinstance(_target,  Unset):
            target = UNSET
        else:
            target = AccessibilityFixInputTarget(_target)




        to_px = d.pop("toPx", UNSET)

        value = d.pop("value", UNSET)

        accessibility_fix_input = cls(
            kind=kind,
            mjml=mjml,
            path=path,
            from_=from_,
            headings=headings,
            index=index,
            role=role,
            target=target,
            to_px=to_px,
            value=value,
        )

        return accessibility_fix_input

