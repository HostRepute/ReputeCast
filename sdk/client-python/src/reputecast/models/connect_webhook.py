from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_webhook_mode import ConnectWebhookMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_registration import ConnectRegistration
  from ..models.connect_self_test import ConnectSelfTest
  from ..models.connect_webhook_test import ConnectWebhookTest





T = TypeVar("T", bound="ConnectWebhook")



@_attrs_define
class ConnectWebhook:
    """ 
        Attributes:
            can_test (bool):
            mode (ConnectWebhookMode):
            needs_secret (bool):
            public_url (str):
            url (str):
            registration (ConnectRegistration | Unset):
            secret_header (str | Unset):
            self_test (ConnectSelfTest | Unset):
            test (ConnectWebhookTest | Unset):
     """

    can_test: bool
    mode: ConnectWebhookMode
    needs_secret: bool
    public_url: str
    url: str
    registration: ConnectRegistration | Unset = UNSET
    secret_header: str | Unset = UNSET
    self_test: ConnectSelfTest | Unset = UNSET
    test: ConnectWebhookTest | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_registration import ConnectRegistration # noqa: PLC0415
        from ..models.connect_self_test import ConnectSelfTest # noqa: PLC0415
        from ..models.connect_webhook_test import ConnectWebhookTest # noqa: PLC0415
        can_test = self.can_test

        mode = self.mode.value

        needs_secret = self.needs_secret

        public_url = self.public_url

        url = self.url

        registration: dict[str, Any] | Unset = UNSET
        if not isinstance(self.registration, Unset):
            registration = self.registration.to_dict()

        secret_header = self.secret_header

        self_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.self_test, Unset):
            self_test = self.self_test.to_dict()

        test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.test, Unset):
            test = self.test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canTest": can_test,
            "mode": mode,
            "needsSecret": needs_secret,
            "publicUrl": public_url,
            "url": url,
        })
        if registration is not UNSET:
            field_dict["registration"] = registration
        if secret_header is not UNSET:
            field_dict["secretHeader"] = secret_header
        if self_test is not UNSET:
            field_dict["selfTest"] = self_test
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_registration import ConnectRegistration # noqa: PLC0415
        from ..models.connect_self_test import ConnectSelfTest # noqa: PLC0415
        from ..models.connect_webhook_test import ConnectWebhookTest # noqa: PLC0415
        d = dict(src_dict)
        can_test = d.pop("canTest")

        mode = ConnectWebhookMode(d.pop("mode"))




        needs_secret = d.pop("needsSecret")

        public_url = d.pop("publicUrl")

        url = d.pop("url")

        _registration = d.pop("registration", UNSET)
        registration: ConnectRegistration | Unset
        if isinstance(_registration,  Unset):
            registration = UNSET
        else:
            registration = ConnectRegistration.from_dict(_registration)




        secret_header = d.pop("secretHeader", UNSET)

        _self_test = d.pop("selfTest", UNSET)
        self_test: ConnectSelfTest | Unset
        if isinstance(_self_test,  Unset):
            self_test = UNSET
        else:
            self_test = ConnectSelfTest.from_dict(_self_test)




        _test = d.pop("test", UNSET)
        test: ConnectWebhookTest | Unset
        if isinstance(_test,  Unset):
            test = UNSET
        else:
            test = ConnectWebhookTest.from_dict(_test)




        connect_webhook = cls(
            can_test=can_test,
            mode=mode,
            needs_secret=needs_secret,
            public_url=public_url,
            url=url,
            registration=registration,
            secret_header=secret_header,
            self_test=self_test,
            test=test,
        )

        return connect_webhook

