from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_alert_action_kind import AutomationAlertActionKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationAlertAction")



@_attrs_define
class AutomationAlertAction:
    """ 
        Attributes:
            kind (AutomationAlertActionKind):
            label (str):  Example: Open the step.
            node_id (str | Unset):
     """

    kind: AutomationAlertActionKind
    label: str
    node_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        node_id = self.node_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if node_id is not UNSET:
            field_dict["nodeId"] = node_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = AutomationAlertActionKind(d.pop("kind"))




        label = d.pop("label")

        node_id = d.pop("nodeId", UNSET)

        automation_alert_action = cls(
            kind=kind,
            label=label,
            node_id=node_id,
        )

        return automation_alert_action

