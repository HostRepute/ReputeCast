from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_incident_link_kind import DeliverabilityIncidentLinkKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityIncidentLink")



@_attrs_define
class DeliverabilityIncidentLink:
    """ 
        Attributes:
            kind (DeliverabilityIncidentLinkKind):
            label (str):
            id (str | Unset):
     """

    kind: DeliverabilityIncidentLinkKind
    label: str
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = DeliverabilityIncidentLinkKind(d.pop("kind"))




        label = d.pop("label")

        id = d.pop("id", UNSET)

        deliverability_incident_link = cls(
            kind=kind,
            label=label,
            id=id,
        )

        return deliverability_incident_link

