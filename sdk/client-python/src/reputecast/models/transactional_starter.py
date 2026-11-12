from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_starter_kind import TransactionalStarterKind
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_test_data import TransactionalTestData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalStarter")



@_attrs_define
class TransactionalStarter:
    """ 
        Attributes:
            attachment (bool): Usually sent with an attachment (a PDF invoice)
            description (str):
            key (str):  Example: password-reset.
            kind (TransactionalStarterKind): The kind of category it belongs in
            name (str):  Example: Password reset.
            preheader (str):
            subject (str):
            test_data (list[TransactionalTestData]):
            variables (list[TransactionalVariable]):
     """

    attachment: bool
    description: str
    key: str
    kind: TransactionalStarterKind
    name: str
    preheader: str
    subject: str
    test_data: list[TransactionalTestData]
    variables: list[TransactionalVariable]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        attachment = self.attachment

        description = self.description

        key = self.key

        kind = self.kind.value

        name = self.name

        preheader = self.preheader

        subject = self.subject

        test_data = []
        for test_data_item_data in self.test_data:
            test_data_item = test_data_item_data.to_dict()
            test_data.append(test_data_item)



        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attachment": attachment,
            "description": description,
            "key": key,
            "kind": kind,
            "name": name,
            "preheader": preheader,
            "subject": subject,
            "testData": test_data,
            "variables": variables,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        attachment = d.pop("attachment")

        description = d.pop("description")

        key = d.pop("key")

        kind = TransactionalStarterKind(d.pop("kind"))




        name = d.pop("name")

        preheader = d.pop("preheader")

        subject = d.pop("subject")

        test_data = []
        _test_data = d.pop("testData")
        for test_data_item_data in (_test_data):
            test_data_item = TransactionalTestData.from_dict(test_data_item_data)



            test_data.append(test_data_item)


        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = TransactionalVariable.from_dict(variables_item_data)



            variables.append(variables_item)


        transactional_starter = cls(
            attachment=attachment,
            description=description,
            key=key,
            kind=kind,
            name=name,
            preheader=preheader,
            subject=subject,
            test_data=test_data,
            variables=variables,
        )

        return transactional_starter

