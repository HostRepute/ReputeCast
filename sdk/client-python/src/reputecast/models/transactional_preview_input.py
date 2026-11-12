from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_address import TransactionalAddress
  from ..models.transactional_preview_input_data import TransactionalPreviewInputData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalPreviewInput")



@_attrs_define
class TransactionalPreviewInput:
    """ 
        Attributes:
            data (TransactionalPreviewInputData | Unset): Data to render with instead of a test data set
            html (str | Unset):
            mjml (str | Unset):
            plain_text (str | Unset):
            preheader (str | Unset):
            subject (str | Unset):
            test_data (str | Unset): The name of a test data set
            to (TransactionalAddress | Unset):
            variables (list[TransactionalVariable] | Unset):
            version (str | Unset): draft (default), live, or a version number Example: draft.
     """

    data: TransactionalPreviewInputData | Unset = UNSET
    html: str | Unset = UNSET
    mjml: str | Unset = UNSET
    plain_text: str | Unset = UNSET
    preheader: str | Unset = UNSET
    subject: str | Unset = UNSET
    test_data: str | Unset = UNSET
    to: TransactionalAddress | Unset = UNSET
    variables: list[TransactionalVariable] | Unset = UNSET
    version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_preview_input_data import TransactionalPreviewInputData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        html = self.html

        mjml = self.mjml

        plain_text = self.plain_text

        preheader = self.preheader

        subject = self.subject

        test_data = self.test_data

        to: dict[str, Any] | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.to_dict()

        variables: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.variables, Unset):
            variables = []
            for variables_item_data in self.variables:
                variables_item = variables_item_data.to_dict()
                variables.append(variables_item)



        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if data is not UNSET:
            field_dict["data"] = data
        if html is not UNSET:
            field_dict["html"] = html
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if plain_text is not UNSET:
            field_dict["plainText"] = plain_text
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if subject is not UNSET:
            field_dict["subject"] = subject
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if to is not UNSET:
            field_dict["to"] = to
        if variables is not UNSET:
            field_dict["variables"] = variables
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_preview_input_data import TransactionalPreviewInputData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        _data = d.pop("data", UNSET)
        data: TransactionalPreviewInputData | Unset
        if isinstance(_data,  Unset):
            data = UNSET
        else:
            data = TransactionalPreviewInputData.from_dict(_data)




        html = d.pop("html", UNSET)

        mjml = d.pop("mjml", UNSET)

        plain_text = d.pop("plainText", UNSET)

        preheader = d.pop("preheader", UNSET)

        subject = d.pop("subject", UNSET)

        test_data = d.pop("testData", UNSET)

        _to = d.pop("to", UNSET)
        to: TransactionalAddress | Unset
        if isinstance(_to,  Unset):
            to = UNSET
        else:
            to = TransactionalAddress.from_dict(_to)




        _variables = d.pop("variables", UNSET)
        variables: list[TransactionalVariable] | Unset = UNSET
        if _variables is not UNSET:
            variables = []
            for variables_item_data in _variables:
                variables_item = TransactionalVariable.from_dict(variables_item_data)



                variables.append(variables_item)


        version = d.pop("version", UNSET)

        transactional_preview_input = cls(
            data=data,
            html=html,
            mjml=mjml,
            plain_text=plain_text,
            preheader=preheader,
            subject=subject,
            test_data=test_data,
            to=to,
            variables=variables,
            version=version,
        )

        return transactional_preview_input

