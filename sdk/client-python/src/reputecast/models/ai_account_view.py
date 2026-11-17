from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_account_view_provider import AIAccountViewProvider
from ..models.ai_account_view_status import AIAccountViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.ai_key_view import AIKeyView
  from ..models.ai_model_view import AIModelView
  from ..models.ai_test_view import AITestView





T = TypeVar("T", bound="AIAccountView")



@_attrs_define
class AIAccountView:
    """ 
        Attributes:
            connected_at (datetime.datetime):
            demo (bool):
            id (str):
            key (AIKeyView):
            models (list[AIModelView]):
            primary (bool):
            private_network (bool):
            provider (AIAccountViewProvider):
            provider_name (str):
            status (AIAccountViewStatus):
            api_version (str | Unset):
            base_url (str | Unset):
            connected_by (str | Unset):
            label (str | Unset):
            last_test (AITestView | Unset):
            preset (str | Unset):
            status_at (datetime.datetime | Unset):
            status_detail (str | Unset):
     """

    connected_at: datetime.datetime
    demo: bool
    id: str
    key: AIKeyView
    models: list[AIModelView]
    primary: bool
    private_network: bool
    provider: AIAccountViewProvider
    provider_name: str
    status: AIAccountViewStatus
    api_version: str | Unset = UNSET
    base_url: str | Unset = UNSET
    connected_by: str | Unset = UNSET
    label: str | Unset = UNSET
    last_test: AITestView | Unset = UNSET
    preset: str | Unset = UNSET
    status_at: datetime.datetime | Unset = UNSET
    status_detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_key_view import AIKeyView # noqa: PLC0415
        from ..models.ai_model_view import AIModelView # noqa: PLC0415
        from ..models.ai_test_view import AITestView # noqa: PLC0415
        connected_at = self.connected_at.isoformat()

        demo = self.demo

        id = self.id

        key = self.key.to_dict()

        models = []
        for models_item_data in self.models:
            models_item = models_item_data.to_dict()
            models.append(models_item)



        primary = self.primary

        private_network = self.private_network

        provider = self.provider.value

        provider_name = self.provider_name

        status = self.status.value

        api_version = self.api_version

        base_url = self.base_url

        connected_by = self.connected_by

        label = self.label

        last_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_test, Unset):
            last_test = self.last_test.to_dict()

        preset = self.preset

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()

        status_detail = self.status_detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connectedAt": connected_at,
            "demo": demo,
            "id": id,
            "key": key,
            "models": models,
            "primary": primary,
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
        if label is not UNSET:
            field_dict["label"] = label
        if last_test is not UNSET:
            field_dict["lastTest"] = last_test
        if preset is not UNSET:
            field_dict["preset"] = preset
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at
        if status_detail is not UNSET:
            field_dict["statusDetail"] = status_detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_key_view import AIKeyView # noqa: PLC0415
        from ..models.ai_model_view import AIModelView # noqa: PLC0415
        from ..models.ai_test_view import AITestView # noqa: PLC0415
        d = dict(src_dict)
        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        demo = d.pop("demo")

        id = d.pop("id")

        key = AIKeyView.from_dict(d.pop("key"))




        models = []
        _models = d.pop("models")
        for models_item_data in (_models):
            models_item = AIModelView.from_dict(models_item_data)



            models.append(models_item)


        primary = d.pop("primary")

        private_network = d.pop("privateNetwork")

        provider = AIAccountViewProvider(d.pop("provider"))




        provider_name = d.pop("providerName")

        status = AIAccountViewStatus(d.pop("status"))




        api_version = d.pop("apiVersion", UNSET)

        base_url = d.pop("baseUrl", UNSET)

        connected_by = d.pop("connectedBy", UNSET)

        label = d.pop("label", UNSET)

        _last_test = d.pop("lastTest", UNSET)
        last_test: AITestView | Unset
        if isinstance(_last_test,  Unset):
            last_test = UNSET
        else:
            last_test = AITestView.from_dict(_last_test)




        preset = d.pop("preset", UNSET)

        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        status_detail = d.pop("statusDetail", UNSET)

        ai_account_view = cls(
            connected_at=connected_at,
            demo=demo,
            id=id,
            key=key,
            models=models,
            primary=primary,
            private_network=private_network,
            provider=provider,
            provider_name=provider_name,
            status=status,
            api_version=api_version,
            base_url=base_url,
            connected_by=connected_by,
            label=label,
            last_test=last_test,
            preset=preset,
            status_at=status_at,
            status_detail=status_detail,
        )

        return ai_account_view

