from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationWorkflowRef")



@_attrs_define
class IntegrationWorkflowRef:
    """ 
        Attributes:
            last_run_at (datetime.datetime):
            name (str):
            unnamed (bool):
     """

    last_run_at: datetime.datetime
    name: str
    unnamed: bool





    def to_dict(self) -> dict[str, Any]:
        last_run_at = self.last_run_at.isoformat()

        name = self.name

        unnamed = self.unnamed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lastRunAt": last_run_at,
            "name": name,
            "unnamed": unnamed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        last_run_at = datetime.datetime.fromisoformat(d.pop("lastRunAt"))




        name = d.pop("name")

        unnamed = d.pop("unnamed")

        integration_workflow_ref = cls(
            last_run_at=last_run_at,
            name=name,
            unnamed=unnamed,
        )

        return integration_workflow_ref

