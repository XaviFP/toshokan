// Helps building SQL query strings by generating a positional argument for every parameter added
pub struct Argumenter<T> {
    pub values: Vec<T>,
}

impl<T> Argumenter<T> {
    pub fn add(&mut self, v: T) -> String {
        self.values.push(v);
        format!("${}", self.values.len())
    }

    pub fn values(&self) -> &Vec<T> {
        &self.values
    }
}

#[test]
fn number_and_order_of_positional_args_is_correct() {
    let mut arger = Argumenter {
        values: Vec::<String>::new(),
    };
    let dollar_1 = arger.add("first".to_owned());
    let dollar_2 = arger.add("second".to_owned());
    let dollar_3 = arger.add("third".to_owned());

    assert_eq!(vec!["$1", "$2", "$3"], vec![dollar_1, dollar_2, dollar_3]);

    let values = arger.values();
    assert_eq!(3, values.len());
    assert_eq!(
        &vec!["first".to_owned(), "second".to_owned(), "third".to_owned()],
        values
    );
}
