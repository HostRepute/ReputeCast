from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationRejectedCall")



@_attrs_define
class IntegrationRejectedCall:
    """ 
        Attributes:
            at (datetime.datetime):
            code (str):
            status (int):
            workflow (str):
     """

    at: datetime.datetime
    code: str
    status: int
    workflow: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        code = self.code

        status = self.status

        workflow = self.workflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "code": code,
            "status": status,
            "workflow": workflow,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        code = d.pop("code")

        status = d.pop("status")

        workflow = d.pop("workflow")

        integration_rejected_call = cls(
            at=at,
            code=code,
            status=status,
            workflow=workflow,
        )

        return integration_rejected_call

