from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeIntegrationsWebhooks")



@_attrs_define
class HostReputeIntegrationsWebhooks:
    """ 
        Attributes:
            endpoints (int):
            failing (int):
     """

    endpoints: int
    failing: int





    def to_dict(self) -> dict[str, Any]:
        endpoints = self.endpoints

        failing = self.failing


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "endpoints": endpoints,
            "failing": failing,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        endpoints = d.pop("endpoints")

        failing = d.pop("failing")

        host_repute_integrations_webhooks = cls(
            endpoints=endpoints,
            failing=failing,
        )

        return host_repute_integrations_webhooks

