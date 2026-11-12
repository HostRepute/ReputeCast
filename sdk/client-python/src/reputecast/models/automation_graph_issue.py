from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_graph_issue_severity import AutomationGraphIssueSeverity
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationGraphIssue")



@_attrs_define
class AutomationGraphIssue:
    """ 
        Attributes:
            code (str):
            message (str):
            severity (AutomationGraphIssueSeverity):
            edge_id (str | Unset):
            node_id (str | Unset):
     """

    code: str
    message: str
    severity: AutomationGraphIssueSeverity
    edge_id: str | Unset = UNSET
    node_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        message = self.message

        severity = self.severity.value

        edge_id = self.edge_id

        node_id = self.node_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "message": message,
            "severity": severity,
        })
        if edge_id is not UNSET:
            field_dict["edgeId"] = edge_id
        if node_id is not UNSET:
            field_dict["nodeId"] = node_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        message = d.pop("message")

        severity = AutomationGraphIssueSeverity(d.pop("severity"))




        edge_id = d.pop("edgeId", UNSET)

        node_id = d.pop("nodeId", UNSET)

        automation_graph_issue = cls(
            code=code,
            message=message,
            severity=severity,
            edge_id=edge_id,
            node_id=node_id,
        )

        return automation_graph_issue

