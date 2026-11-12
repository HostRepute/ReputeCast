from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_connection_view_provider import AIConnectionViewProvider
from ..models.ai_connection_view_status import AIConnectionViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.ai_cap_view import AICapView
  from ..models.ai_key_view import AIKeyView
  from ..models.ai_prices_view import AIPricesView
  from ..models.ai_test_view import AITestView





T = TypeVar("T", bound="AIConnectionView")



@_attrs_define
class AIConnectionView:
    """ 
        Attributes:
            cap (AICapView):
            connected_at (datetime.datetime):
            demo (bool):
            image_input (bool):
            key (AIKeyView):
            model (str):
            model_name (str):
            prices (AIPricesView):
            private_network (bool):
            provider (AIConnectionViewProvider):
            provider_name (str):
            status (AIConnectionViewStatus):
            api_version (str | Unset):
            base_url (str | Unset):
            connected_by (str | Unset):
            deployment (str | Unset):
            label (str | Unset):
            last_test (AITestView | Unset):
            status_at (datetime.datetime | Unset):
            status_detail (str | Unset):
     """

    cap: AICapView
    connected_at: datetime.datetime
    demo: bool
    image_input: bool
    key: AIKeyView
    model: str
    model_name: str
    prices: AIPricesView
    private_network: bool
    provider: AIConnectionViewProvider
    provider_name: str
    status: AIConnectionViewStatus
    api_version: str | Unset = UNSET
    base_url: str | Unset = UNSET
    connected_by: str | Unset = UNSET
    deployment: str | Unset = UNSET
    label: str | Unset = UNSET
    last_test: AITestView | Unset = UNSET
    status_at: datetime.datetime | Unset = UNSET
    status_detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_cap_view import AICapView # noqa: PLC0415
        from ..models.ai_key_view import AIKeyView # noqa: PLC0415
        from ..models.ai_prices_view import AIPricesView # noqa: PLC0415
        from ..models.ai_test_view import AITestView # noqa: PLC0415
        cap = self.cap.to_dict()

        connected_at = self.connected_at.isoformat()

        demo = self.demo

        image_input = self.image_input

        key = self.key.to_dict()

        model = self.model

        model_name = self.model_name

        prices = self.prices.to_dict()

        private_network = self.private_network

        provider = self.provider.value

        provider_name = self.provider_name

        status = self.status.value

        api_version = self.api_version

        base_url = self.base_url

        connected_by = self.connected_by

        deployment = self.deployment

        label = self.label

        last_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_test, Unset):
            last_test = self.last_test.to_dict()

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()

        status_detail = self.status_detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "connectedAt": connected_at,
            "demo": demo,
            "imageInput": image_input,
            "key": key,
            "model": model,
            "modelName": model_name,
            "prices": prices,
            "privateNetwork": private_network,
            "provider": provider,
            "providerName": provider_name,
            "status": status,
        })
        if api_version is not UNSET:
            field_dict["apiVersion"] = api_version
        if base_url is not UNSET:
            field_dict["baseUrl"] = base_url
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if deployment is not UNSET:
            field_dict["deployment"] = deployment
        if label is not UNSET:
            field_dict["label"] = label
        if last_test is not UNSET:
            field_dict["lastTest"] = last_test
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at
        if status_detail is not UNSET:
            field_dict["statusDetail"] = status_detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_cap_view import AICapView # noqa: PLC0415
        from ..models.ai_key_view import AIKeyView # noqa: PLC0415
        from ..models.ai_prices_view import AIPricesView # noqa: PLC0415
        from ..models.ai_test_view import AITestView # noqa: PLC0415
        d = dict(src_dict)
        cap = AICapView.from_dict(d.pop("cap"))




        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        demo = d.pop("demo")

        image_input = d.pop("imageInput")

        key = AIKeyView.from_dict(d.pop("key"))




        model = d.pop("model")

        model_name = d.pop("modelName")

        prices = AIPricesView.from_dict(d.pop("prices"))




        private_network = d.pop("privateNetwork")

        provider = AIConnectionViewProvider(d.pop("provider"))




        provider_name = d.pop("providerName")

        status = AIConnectionViewStatus(d.pop("status"))




        api_version = d.pop("apiVersion", UNSET)

        base_url = d.pop("baseUrl", UNSET)

        connected_by = d.pop("connectedBy", UNSET)

        deployment = d.pop("deployment", UNSET)

        label = d.pop("label", UNSET)

        _last_test = d.pop("lastTest", UNSET)
        last_test: AITestView | Unset
        if isinstance(_last_test,  Unset):
            last_test = UNSET
        else:
            last_test = AITestView.from_dict(_last_test)




        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        status_detail = d.pop("statusDetail", UNSET)

        ai_connection_view = cls(
            cap=cap,
            connected_at=connected_at,
            demo=demo,
            image_input=image_input,
            key=key,
            model=model,
            model_name=model_name,
            prices=prices,
            private_network=private_network,
            provider=provider,
            provider_name=provider_name,
            status=status,
            api_version=api_version,
            base_url=base_url,
            connected_by=connected_by,
            deployment=deployment,
            label=label,
            last_test=last_test,
            status_at=status_at,
            status_detail=status_detail,
        )

        return ai_connection_view

