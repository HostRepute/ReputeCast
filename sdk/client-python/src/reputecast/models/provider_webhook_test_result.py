from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderWebhookTestResult")



@_attrs_define
class ProviderWebhookTestResult:
    """ 
        Attributes:
            endpoint_detail (str):
            endpoint_ok (bool):
            endpoint_status (int):
            provider_test (bool):
            provider_test_detail (str | Unset):
     """

    endpoint_detail: str
    endpoint_ok: bool
    endpoint_status: int
    provider_test: bool
    provider_test_detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        endpoint_detail = self.endpoint_detail

        endpoint_ok = self.endpoint_ok

        endpoint_status = self.endpoint_status

        provider_test = self.provider_test

        provider_test_detail = self.provider_test_detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "endpointDetail": endpoint_detail,
            "endpointOk": endpoint_ok,
            "endpointStatus": endpoint_status,
            "providerTest": provider_test,
        })
        if provider_test_detail is not UNSET:
            field_dict["providerTestDetail"] = provider_test_detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        endpoint_detail = d.pop("endpointDetail")

        endpoint_ok = d.pop("endpointOk")

        endpoint_status = d.pop("endpointStatus")

        provider_test = d.pop("providerTest")

        provider_test_detail = d.pop("providerTestDetail", UNSET)

        provider_webhook_test_result = cls(
            endpoint_detail=endpoint_detail,
            endpoint_ok=endpoint_ok,
            endpoint_status=endpoint_status,
            provider_test=provider_test,
            provider_test_detail=provider_test_detail,
        )

        return provider_webhook_test_result

